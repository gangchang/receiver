package options

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"

	"github.com/rancher/webhook-receiver/pkg/providers"
	"github.com/rancher/webhook-receiver/pkg/providers/alibaba"
	"github.com/rancher/webhook-receiver/pkg/providers/dingtalk"
)

var (
	mut       sync.RWMutex
	receivers map[string]providers.Receiver
	senders   map[string]providers.Sender
)

// when occur error, it will panic directly
func Init(configPath string) {
	dir := filepath.Dir(configPath)
	log.Infof("config dir:%s\n", dir)
	name := filepath.Base(configPath)
	log.Infof("config name:%s\n", name)
	viperConfigName := strings.TrimRight(name, ".yaml")
	viper.AddConfigPath(dir)
	viper.SetConfigName(viperConfigName)
	viper.SetConfigType("yaml")

	updateMemoryConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		updateMemoryConfig()
	})

	go viper.WatchConfig()
}

func GetReceiverAndSender(receiverName string) (providers.Receiver, providers.Sender, error) {
	mut.RLock()
	defer mut.RUnlock()

	receiver, exists := receivers[receiverName]
	if !exists {
		return providers.Receiver{}, nil, fmt.Errorf("error, receiver:%s is not exists\n", receiverName)
	}

	sender, exists := senders[receiver.Provider]
	if !exists {
		return providers.Receiver{}, nil, fmt.Errorf("error, provider:%s is not exists\n", receiver.Provider)
	}

	return receiver, sender, nil
}

func updateMemoryConfig() {
	if err := viper.ReadInConfig(); err != nil {
		log.Errorf("read config err:%v", err)
		return
	}

	receiversMap := viper.GetStringMap("receivers")
	updateReceivers := make(map[string]providers.Receiver)
	for k, v := range receiversMap {
		receiver := providers.Receiver{}
		if err := convertInterfaceToStruct(v, &receiver); err != nil {
			log.Errorf("parse receiver:%s to struct err:%v", k, err)
			return
		}
		updateReceivers[k] = receiver
	}

	providersMap := viper.GetStringMap("providers")
	updateSenders := make(map[string]providers.Sender)
	for k, v := range providersMap {
		creator, err := getProviderCreator(k)
		if err != nil {
			log.Errorf("update config err:%v", err)
			return
		}
		optMap := make(map[string]string)
		if err := convertInterfaceToStruct(v, &optMap); err != nil {
			log.Errorf("parse provider:%s err:%v", k, err)
			return
		}
		sender, err := creator(optMap)
		if err != nil {
			log.Errorf("update config err:%v", err)
			return
		}
		updateSenders[k] = sender
	}

	// replace
	mut.Lock()
	defer mut.Unlock()
	receivers = updateReceivers
	senders = updateSenders
	log.Info("update config success")
}

func getProviderCreator(name string) (providers.Creator, error) {
	switch name {
	case alibaba.Name:
		return alibaba.New, nil
	case dingtalk.Name:
		return dingtalk.New, nil
	default:
		return nil, errors.New(fmt.Sprintf("provider %s is not support", name))
	}
}

// for yaml parse
type option struct {
	Providers map[string]map[string]string
	Receivers []providers.Receiver
}

func newOption(data []byte) (option, error) {
	opt := option{}
	if err := yaml.Unmarshal(data, &opt); err != nil {
		return option{}, err
	}

	return opt, nil
}

func convertInterfaceToStruct(inter interface{}, s interface{}) error {
	byteData, err := yaml.Marshal(inter)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(byteData, s); err != nil {
		return err
	}

	return nil
}
