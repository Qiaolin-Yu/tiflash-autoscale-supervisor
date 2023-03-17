package main

import (
	"fmt"
	"log"
	"os"
)

var (
	learnerConfigTemplateFilename   = "conf/tiflash-learner-templete.toml"
	learnerConfigFilename           = "conf/tiflash-learner.toml"
	tiFlashConfigTemplateFilename   = "conf/tiflash-templete.toml"
	tiFlashPreprocessedConfigBuffer = ""
	// tiFlashPreprocessedConfigFilename = "conf/tiflash-preprocessed.toml"
)

func RenderTiFlashConf(targetTiFlashConfigFilename string, tidbStatusAddr string, pdAddr string, tenantName string) error {
	// tiFlashPreprocessedConfigFile := tiFlashPreprocessedConfigBuffer
	// if err != nil {
	// log.Printf("could not read tiflash-preprocessed config file %v: %v", tiFlashPreprocessedConfigFilename, err)
	// return err
	// }
	tiFlashPreprocessedConfig := string(tiFlashPreprocessedConfigBuffer)
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

func InitTiFlashConf(localIp string) error {
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
	tiFlashConfig := fmt.Sprintf(tiFlashConfigTemplate, "%v", localIp, "%v", "%v")
	tiFlashPreprocessedConfigBuffer = tiFlashConfig
	// _, err = tiFlashConfigFile.WriteString(tiFlashConfig)
	// if err != nil {
	// log.Printf("could not write tiflash-preprocessed config file %v: %v", tiFlashPreprocessedConfigFilename, err)
	// return err
	// }
	return nil
}
