package main

import (
	"fmt"
	"log"
	"os"
)

var (
	// ConfPath                        = "conf"
	tiFlashPreprocessedConfigBuffer = ""
	ver2confBuffer                  = make(map[string]string)
	DefaultVersion                  = "" // "" means default tiflash version
	// tiFlashPreprocessedConfigFilename = "conf/tiflash-preprocessed.toml"
)

func GetConfPath(ver string) string {
	if ver == "s3" {
		return "conf/s3"
	} else {
		return "conf"
	}
}

func GenLearnerConfigTemplateFilename(ver string) string {
	return GetConfPath(ver) + "/tiflash-learner-templete.toml"
}

func GenLearnerConfigFilename() string {
	return GetConfPath("") + "/tiflash-learner.toml"
}

func GenTiFlashConfigTemplateFilename(ver string) string {
	return GetConfPath(ver) + "/tiflash-templete.toml"
}

func SavePreprocessedConfigBuffer(ver string, tiFlashConfig string) {
	ver2confBuffer[ver] = tiFlashConfig
}

func CheckPreprocessedConfigBufferExist(ver string) bool {
	_, ok := ver2confBuffer[ver]
	return ok
}

func LoadPreprocessedConfigBuffer(ver string) string {
	return ver2confBuffer[ver]
}

// func RenderTiFlashConf(targetTiFlashConfigFilename string, tidbStatusAddr string, pdAddr string, tenantName string) error {
// 	return RenderTiFlashConfWithVer(targetTiFlashConfigFilename, tidbStatusAddr, pdAddr, tenantName, "")
// }

func RenderTiFlashConf(targetTiFlashConfigFilename string, tidbStatusAddr string, pdAddr string, tenantName string, ver string) error {
	// tiFlashPreprocessedConfigFile := tiFlashPreprocessedConfigBuffer
	// if err != nil {
	// log.Printf("could not read tiflash-preprocessed config file %v: %v", tiFlashPreprocessedConfigFilename, err)
	// return err
	// }
	tiFlashPreprocessedConfig := string(LoadPreprocessedConfigBuffer(ver))
	fixPoolConfItem := ""
	// if tenantName == "fixpool-use-autoscaler-false" {
	// 	fixPoolConfItem = "use_autoscaler = false"
	// } else if tenantName == "fixpool-use-autoscaler-true" {
	// 	fixPoolConfItem = "use_autoscaler = true"
	// }
	tiFlashConfig := fmt.Sprintf(tiFlashPreprocessedConfig, tenantName, fixPoolConfItem, pdAddr)
	tiFlashConfigFile, err := os.Create(targetTiFlashConfigFilename)
	defer tiFlashConfigFile.Close()
	if err != nil {
		log.Printf("could not create tiflash config file %v: %v", targetTiFlashConfigFilename, err)
		return err
	}
	_, err = tiFlashConfigFile.WriteString(tiFlashConfig)
	if err != nil {
		log.Printf("could not write tiflash config file %v: %v", targetTiFlashConfigFilename, err)
		return err
	}
	return nil
}

// func InitTiFlashConf(localIp string) error {
// 	return InitTiFlashConfWithVer(localIp, "")
// }

func InitTiFlashConf(localIp string, ver string) error {
	if CheckPreprocessedConfigBufferExist(ver) {
		// has inited
		return nil
	}
	learnerConfigTemplateFilename := GenLearnerConfigTemplateFilename(ver)
	learnerConfigFilename := GenLearnerConfigFilename()
	tiFlashConfigTemplateFilename := GenTiFlashConfigTemplateFilename(ver)
	learnerConfigTemplateFile, err := os.ReadFile(learnerConfigTemplateFilename)
	learnerConfigTemplate := string(learnerConfigTemplateFile)
	if err != nil {
		log.Printf("could not read tiflash-learner config templete %v: %v", learnerConfigTemplateFilename, err)
		return err
	}
	learnerConfigFile, err := os.Create(learnerConfigFilename)
	defer learnerConfigFile.Close()
	if err != nil {
		log.Printf("could not create tiflash-learner config file %v: %v", learnerConfigFilename, err)
		return err
	}
	learnerConfig := fmt.Sprintf(learnerConfigTemplate, localIp, localIp, localIp)
	_, err = learnerConfigFile.WriteString(learnerConfig)
	if err != nil {
		log.Printf("could not write tiflash-learner config file %v: %v", learnerConfigFilename, err)
		return err
	}

	tiFlashConfigTemplateFile, err := os.ReadFile(tiFlashConfigTemplateFilename)
	tiFlashConfigTemplate := string(tiFlashConfigTemplateFile)
	if err != nil {
		log.Printf("could not read tiflash-preprocessed config templete %v: %v", tiFlashConfigTemplateFilename, err)
		return err
	}
	// tiFlashConfigFile, err := os.Create(tiFlashPreprocessedConfigFilename)
	// defer tiFlashConfigFile.Close()
	// if err != nil {
	// 	log.Printf("could not create tiflash-preprocessed config file %v: %v", tiFlashPreprocessedConfigFilename, err)
	// 	return err
	// }
	tiFlashConfig := fmt.Sprintf(tiFlashConfigTemplate, "%v", localIp, "%v", "%v", PathOfTiflashData, PathOfTiflashCache, CapicityOfTiflashCache)
	SavePreprocessedConfigBuffer(ver, tiFlashConfig)
	// tiFlashPreprocessedConfigBuffer = tiFlashConfig
	// _, err = tiFlashConfigFile.WriteString(tiFlashConfig)
	// if err != nil {
	// log.Printf("could not write tiflash-preprocessed config file %v: %v", tiFlashPreprocessedConfigFilename, err)
	// return err
	// }
	return nil
}
