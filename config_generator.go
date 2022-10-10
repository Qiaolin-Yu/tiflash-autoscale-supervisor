package main

import (
	"fmt"
	"log"
	"os"
)

var (
	learnerConfigTemplateFilename     = "conf/tiflash-learner-templete.toml"
	learnerConfigFilename             = "conf/tiflash-learner.toml"
	tiFlashConfigTemplateFilename     = "conf/tiflash-templete.toml"
	tiFlashPreprocessedConfigFilename = "conf/tiflash-preprocessed.toml"
)

func RenderTiFlashConf(tiFlashConfigFilename string, tidbStatusAddr string, pdAddr string) error {
	tiFlashPreprocessedConfigFile, err := os.ReadFile(tiFlashPreprocessedConfigFilename)
	if err != nil {
		log.Printf("could not read tiflash-preprocessed config file %v: %v", tiFlashPreprocessedConfigFilename, err)
		return err
	}
	tiFlashPreprocessedConfig := string(tiFlashPreprocessedConfigFile)
	tiFlashConfig := fmt.Sprintf(tiFlashPreprocessedConfig, tidbStatusAddr, pdAddr)
	tiFlashConfigFile, err := os.Create(tiFlashConfigFilename)
	defer tiFlashConfigFile.Close()
	if err != nil {
		log.Printf("could not create tiflash config file %v: %v", tiFlashConfigFilename, err)
		return err
	}
	_, err = tiFlashConfigFile.WriteString(tiFlashConfig)
	if err != nil {
		log.Printf("could not write tiflash config file %v: %v", tiFlashConfigFilename, err)
		return err
	}
	return nil
}

func InitTiFlashConf() error {
	localIp := os.Getenv("POD_IP")
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
	tiFlashConfigFile, err := os.Create(tiFlashPreprocessedConfigFilename)
	defer tiFlashConfigFile.Close()
	if err != nil {
		log.Printf("could not create tiflash-preprocessed config file %v: %v", tiFlashPreprocessedConfigFilename, err)
		return err
	}
	tiFlashConfig := fmt.Sprintf(tiFlashConfigTemplate, localIp, "%v", "%v")
	_, err = tiFlashConfigFile.WriteString(tiFlashConfig)
	if err != nil {
		log.Printf("could not write tiflash-preprocessed config file %v: %v", tiFlashPreprocessedConfigFilename, err)
		return err
	}
	return nil
}
