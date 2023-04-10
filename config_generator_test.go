package main

import (
	"os"
	"testing"

	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/assert"
)

func TestMissingValue(t *testing.T) {
	//Detect missing value in tiflash learner config
	err := os.Setenv("POD_IP", "127.1.1.3")
	assert.NoError(t, err)
	LocalPodIp = os.Getenv("POD_IP")
	err = InitTiFlashConf(LocalPodIp, DefaultVersion)
	assert.NoError(t, err)
	learnerConfigTemplateFile, err := os.ReadFile(GenLearnerConfigTemplateFilename(DefaultVersion))
	assert.NoError(t, err)
	learnerConfigTemplate := string(learnerConfigTemplateFile)
	assert.NotContains(t, learnerConfigTemplate, "MISSING")

	//Detect missing value in tiflash config
	err = RenderTiFlashConf("conf/tiflash.toml", "123.123.123.123:1000", "179.1.1.1:2000", "tenant-test", DefaultVersion)
	assert.NoError(t, err)
	tiflashConfigTemplateFile, err := os.ReadFile("conf/tiflash.toml")
	assert.NoError(t, err)
	tiflashConfigTemplate := string(tiflashConfigTemplateFile)
	assert.NotContains(t, tiflashConfigTemplate, "MISSING")
}

func TestTiFlashConfGenerator(t *testing.T) {
	err := os.Setenv("POD_IP", "127.1.1.3")
	assert.NoError(t, err)
	LocalPodIp = os.Getenv("POD_IP")
	err = InitTiFlashConf(LocalPodIp, DefaultVersion)
	assert.NoError(t, err)
	err = InitTiFlashConf(LocalPodIp, DefaultVersion)
	assert.NoError(t, err)
	err = InitTiFlashConf(LocalPodIp, DefaultVersion)
	assert.NoError(t, err)
	config, err := toml.LoadFile(GenLearnerConfigFilename())
	assert.NoError(t, err)
	configString, err := config.ToTomlString()
	assert.NoError(t, err)
	assert.NotContains(t, configString, "MISSING")
	assert.Equal(t, config.Get("server.advertise-addr").(string), "127.1.1.3:20170")
	assert.Equal(t, config.Get("server.advertise-status-addr").(string), "127.1.1.3:20292")
	assert.Equal(t, config.Get("server.engine-addr").(string), "127.1.1.3:3930")

	// Test1 of RenderTiFlashConf
	err = RenderTiFlashConf("conf/tiflash.toml", "123.123.123.123:1000", "179.1.1.1:2000", "tenant-test", DefaultVersion)
	assert.NoError(t, err)
	err = RenderTiFlashConf("conf/tiflash.toml", "123.123.123.123:1000", "179.1.1.1:2000", "tenant-test", DefaultVersion)
	assert.NoError(t, err)
	err = RenderTiFlashConf("conf/tiflash.toml", "123.123.123.123:1000", "179.1.1.1:2000", "tenant-test", DefaultVersion)
	assert.NoError(t, err)
	config, err = toml.LoadFile("conf/tiflash.toml")
	assert.NoError(t, err)
	configString, err = config.ToTomlString()
	assert.NoError(t, err)
	assert.NotContains(t, configString, "MISSING")
	assert.Equal(t, config.Get("flash.service_addr").(string), "127.1.1.3:3930")
	assert.Equal(t, config.Get("cluster.cluster_id").(string), "tenant-test")
	assert.Equal(t, config.Get("raft.pd_addr").(string), "179.1.1.1:2000")
	assert.Equal(t, config.Get("flash.use_autoscaler"), true)
	assert.Equal(t, config.Get("flash.use_autoscaler_without_s3"), true)
	assert.Equal(t, config.Get("profiles.default.max_memory_usage_for_all_queries"), 0.9)

	// // Test2 of RenderTiFlashConf
	// err = RenderTiFlashConf("conf/tiflash.toml", "125.125.125.125:1000", "179.2.2.2:2000", "fixpool-use-autoscaler-false")
	// assert.NoError(t, err)
	// config, err = toml.LoadFile("conf/tiflash.toml")
	// assert.NoError(t, err)
	// configString, err = config.ToTomlString()
	// assert.NoError(t, err)
	// assert.NotContains(t, configString, "MISSING")
	// assert.Equal(t, config.Get("flash.service_addr").(string), "127.1.1.3:3930")
	// assert.Equal(t, config.Get("cluster.cluster_id").(string), "fixpool-use-autoscaler-false")
	// assert.Equal(t, config.Get("raft.pd_addr").(string), "179.2.2.2:2000")
	// assert.Equal(t, config.Get("flash.use_autoscaler").(bool), false)

	// // Test3 of RenderTiFlashConf
	// err = RenderTiFlashConf("conf/tiflash.toml", "255.255.255.255:1000", "179.3.3.3:3000", "fixpool-use-autoscaler-true")
	// assert.NoError(t, err)
	// config, err = toml.LoadFile("conf/tiflash.toml")
	// assert.NoError(t, err)
	// configString, err = config.ToTomlString()
	// assert.NoError(t, err)
	// assert.NotContains(t, configString, "MISSING")
	// assert.Equal(t, config.Get("flash.service_addr").(string), "127.1.1.3:3930")
	// assert.Equal(t, config.Get("cluster.cluster_id").(string), "fixpool-use-autoscaler-true")
	// assert.Equal(t, config.Get("raft.pd_addr").(string), "179.3.3.3:3000")
	// assert.Equal(t, config.Get("flash.use_autoscaler").(bool), true)

}

func TestTiFlashConfGeneratorWithVer(t *testing.T) {
	ver := "s3"
	err := os.Setenv("POD_IP", "127.1.1.3")
	assert.NoError(t, err)
	LocalPodIp = os.Getenv("POD_IP")
	err = InitTiFlashConf(LocalPodIp, ver)
	assert.NoError(t, err)
	err = InitTiFlashConf(LocalPodIp, ver)
	assert.NoError(t, err)
	err = InitTiFlashConf(LocalPodIp, ver)
	assert.NoError(t, err)
	assert.NoError(t, err)
	config, err := toml.LoadFile("conf/tiflash-learner.toml")
	assert.NoError(t, err)
	assert.Equal(t, config.Get("server.advertise-addr").(string), "127.1.1.3:20170")
	assert.Equal(t, config.Get("server.advertise-status-addr").(string), "127.1.1.3:20292")
	assert.Equal(t, config.Get("server.engine-addr").(string), "127.1.1.3:3930")

	// Test1 of RenderTiFlashConf
	err = RenderTiFlashConf("conf/tiflash.toml", "123.123.123.123:1000", "179.1.1.1:2000", "tenant-test", ver)
	assert.NoError(t, err)
	err = RenderTiFlashConf("conf/tiflash.toml", "123.123.123.123:1000", "179.1.1.1:2000", "tenant-test", ver)
	assert.NoError(t, err)
	err = RenderTiFlashConf("conf/tiflash.toml", "123.123.123.123:1000", "179.1.1.1:2000", "tenant-test", ver)
	assert.NoError(t, err)
	config, err = toml.LoadFile("conf/tiflash.toml")
	assert.NoError(t, err)
	assert.Equal(t, config.Get("flash.service_addr").(string), "127.1.1.3:3930")
	assert.Equal(t, config.Get("cluster.cluster_id").(string), "tenant-test")
	assert.Equal(t, config.Get("raft.pd_addr").(string), "179.1.1.1:2000")
	assert.Equal(t, config.Get("flash.use_autoscaler"), true)
	assert.Equal(t, config.Get("flash.use_autoscaler_without_s3"), false)
	assert.Equal(t, config.Get("profiles.default.max_memory_usage_for_all_queries"), 0.9)

	// // Test2 of RenderTiFlashConf
	// err = RenderTiFlashConf("conf/tiflash.toml", "125.125.125.125:1000", "179.2.2.2:2000", "fixpool-use-autoscaler-false")
	// assert.NoError(t, err)
	// config, err = toml.LoadFile("conf/tiflash.toml")
	// assert.NoError(t, err)
	// assert.Equal(t, config.Get("flash.service_addr").(string), "127.1.1.3:3930")
	// assert.Equal(t, config.Get("cluster.cluster_id").(string), "fixpool-use-autoscaler-false")
	// assert.Equal(t, config.Get("raft.pd_addr").(string), "179.2.2.2:2000")
	// assert.Equal(t, config.Get("flash.use_autoscaler").(bool), false)

	// // Test3 of RenderTiFlashConf
	// err = RenderTiFlashConf("conf/tiflash.toml", "255.255.255.255:1000", "179.3.3.3:3000", "fixpool-use-autoscaler-true")
	// assert.NoError(t, err)
	// config, err = toml.LoadFile("conf/tiflash.toml")
	// assert.NoError(t, err)
	// assert.Equal(t, config.Get("flash.service_addr").(string), "127.1.1.3:3930")
	// assert.Equal(t, config.Get("cluster.cluster_id").(string), "fixpool-use-autoscaler-true")
	// assert.Equal(t, config.Get("raft.pd_addr").(string), "179.3.3.3:3000")
	// assert.Equal(t, config.Get("flash.use_autoscaler").(bool), true)
}

func TestHehe(t *testing.T) {
	mp := make(map[string]string)
	mp[""] = "abc"
	v := mp[""]
	assert.Equal(t, v, "abc")
}
