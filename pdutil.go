package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

func PdCtlFindStoreIdFromJsonStr(str string) string {
	var x map[string]interface{}
	err := json.Unmarshal([]byte(str), &x)
	if err != nil {
		return ""
	}
	arr, ok := x["stores"].([]interface{})
	if !ok {
		return ""
	}
	for _, item := range arr {
		jsonmap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		storeMap, ok := jsonmap["store"]
		if !ok {
			continue
		}
		mapPart, ok := storeMap.(map[string]interface{})
		if !ok {
			continue
		}
		// fmt.Println(mapPart)
		storeAddr, ok := mapPart["address"]
		if !ok {
			continue
		}
		storeAddrStr, ok := storeAddr.(string)
		if !ok {
			continue
		}
		addrArr := strings.Split(storeAddrStr, ":")
		if len(addrArr) == 0 {
			continue
		}
		if addrArr[0] == LocalPodIp {
			sid, ok := mapPart["id"].(float64)
			if !ok {
				return ""
			} else {
				return strconv.Itoa(int(sid))
			}
		}
	}
	return ""
}

func PdCtlGetStoreIdsOfUnhealthRNs(str string) []string {

	var x map[string]interface{}
	err := json.Unmarshal([]byte(str), &x)
	if err != nil {
		return nil
	}
	arr, ok := x["stores"].([]interface{})
	if !ok {
		fmt.Println("#2")
		return nil
	}
	retStoreIDs := make([]string, 0, 5)
	for _, item := range arr {
		jsonmap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		storeMap, ok := jsonmap["store"]
		if !ok {
			continue
		}
		mapPart, ok := storeMap.(map[string]interface{})
		if !ok {
			continue
		}
		// fmt.Println(mapPart)

		rawLabelMap, ok := mapPart["labels"]
		if !ok {
			continue
		}
		rawLabels, ok := rawLabelMap.([]interface{})
		if !ok {
			continue
		}
		for _, rawLabel := range rawLabels {
			label, ok := rawLabel.(map[string]interface{})
			if !ok {
				continue
			}
			labelKey, ok := label["key"]
			if !ok {
				continue
			}
			if labelKey == "engine" {
				labelValue, ok := label["value"]
				if !ok || labelValue != "tiflash_mpp" {
					continue
				} else {
					state, ok := mapPart["state_name"]
					if ok && state != "Up" && state != "up" && state != "UP" {
						// record unhealthy RNs from PD
						sid, ok := mapPart["id"].(float64)
						if !ok {
							continue
						} else {
							retStoreIDs = append(retStoreIDs, strconv.Itoa(int(sid)))
						}
					} else {
						continue
					}
				}
			}
		}

	}
	return retStoreIDs
}

func PdCtlRemoveStoreIDsOfUnhealthRNs() error {
	if !NeedPd {
		return nil
	}
	outOfPdctl, err := exec.Command("./bin/pd-ctl", "-u", "http://"+PdAddr, "store").Output()
	if err != nil {
		log.Printf("[error][RemoveStoreIDsOfUnhealthRNs]pd ctl get store error: %v\n", err.Error())
		return err
	}
	sIDs := PdCtlGetStoreIdsOfUnhealthRNs(string(outOfPdctl))
	if sIDs != nil {
		for _, storeID := range sIDs {
			err := PdCtlRemoveStoreIDFromPD(storeID)
			if err != nil {
				continue
			}
		}
	}
	err = PdCtlRemoveTombStonesFromPD()

	return err
}

func PdCtlRemoveStoreIDFromPD(storeID string) error {
	if storeID != "" {
		_, err := exec.Command("./bin/pd-ctl", "-u", "http://"+PdAddr, "store", "delete", storeID).Output()
		if err != nil {
			log.Printf("[error]pd ctl get store error: %v\n", err.Error())
			return err
		}
	}
	return nil
}

func PdCtlRemoveTombStonesFromPD() error {
	_, err := exec.Command("./bin/pd-ctl", "-u", "http://"+PdAddr, "store", "remove-tombstone").Output()
	if err != nil {
		log.Printf("[error]pd ctl get store error: %v\n", err.Error())
	}
	return err
}

func PdCtlNotifyPDForExit() error {
	outOfPdctl, err := exec.Command("./bin/pd-ctl", "-u", "http://"+PdAddr, "store").Output()
	if err != nil {
		log.Printf("[error]pd ctl get store error: %v\n", err.Error())
		return err
	}
	storeID := PdCtlFindStoreIdFromJsonStr(string(outOfPdctl))
	if storeID != "" {
		err := PdCtlRemoveStoreIDFromPD(storeID)
		if err != nil {
			return err
		}
		err = PdCtlRemoveTombStonesFromPD()
		if err != nil {
			return err
		}
	}
	return nil
}
