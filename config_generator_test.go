package main

import (
	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestTiFlashConfGenerator(t *testing.T) {
	err := os.Setenv("POD_IP", "127.1.1.3")
	assert.NoError(t, err)
	err = InitTiFlashConf()
	assert.NoError(t, err)
	config, err := toml.LoadFile("conf/tiflash-learner.toml")
	assert.NoError(t, err)
	assert.Equal(t, config.Get("server.advertise-addr").(string), "127.1.1.3:20170")
	assert.Equal(t, config.Get("server.advertise-status-addr").(string), "127.1.1.3:20292")
	assert.Equal(t, config.Get("server.engine-addr").(string), "127.1.1.3:3930")

	// Test1 of RenderTiFlashConf
	err = RenderTiFlashConf("conf/tiflash.toml", "123.123.123.123:1000", "179.1.1.1:2000", "tenant-test")
	assert.NoError(t, err)
	config, err = toml.LoadFile("conf/tiflash.toml")
	assert.NoError(t, err)
	assert.Equal(t, config.Get("flash.service_addr").(string), "127.1.1.3:3930")
	assert.Equal(t, config.Get("cluster.cluster_id").(string), "tenant-test")
	assert.Equal(t, config.Get("raft.pd_addr").(string), "179.1.1.1:2000")
	assert.Equal(t, config.Get("flash.use_autoscaler"), nil)

	// Test2 of RenderTiFlashConf
	err = RenderTiFlashConf("conf/tiflash.toml", "123.123.123.123:1000", "179.1.1.1:2000", "fixpool-use-autoscaler-false")
	assert.NoError(t, err)
	config, err = toml.LoadFile("conf/tiflash.toml")
	assert.NoError(t, err)
	assert.Equal(t, config.Get("flash.service_addr").(string), "127.1.1.3:3930")
	assert.Equal(t, config.Get("cluster.cluster_id").(string), "fixpool-use-autoscaler-false")
	assert.Equal(t, config.Get("raft.pd_addr").(string), "179.1.1.1:2000")
	assert.Equal(t, config.Get("flash.use_autoscaler").(bool), false)

	// Test3 of RenderTiFlashConf
	err = RenderTiFlashConf("conf/tiflash.toml", "123.123.123.123:1000", "179.1.1.1:2000", "fixpool-use-autoscaler-true")
	assert.NoError(t, err)
	config, err = toml.LoadFile("conf/tiflash.toml")
	assert.NoError(t, err)
	assert.Equal(t, config.Get("flash.service_addr").(string), "127.1.1.3:3930")
	assert.Equal(t, config.Get("cluster.cluster_id").(string), "fixpool-use-autoscaler-true")
	assert.Equal(t, config.Get("raft.pd_addr").(string), "179.1.1.1:2000")
	assert.Equal(t, config.Get("flash.use_autoscaler").(bool), true)
}
