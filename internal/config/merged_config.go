package config

import (
	"errors"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Reads configurations from all specified files using read(),
// and then merges them into a single viper.Viper instance.
func readMerged(files ...string) (*wrapperView, error) {
	if len(files) == 0 {
		return nil, errors.New("no input files specified")
	}

	layers := make([]map[string]interface{}, len(files))

	type change struct {
		idx   int
		layer map[string]interface{}
	}
	changes := make(chan change, len(layers))

	// read files into layers and watch for changes
	for i, f := range files {
		idx := i
		s, err := read(f, nil)
		if err != nil {
			return nil, err
		}
		layers[idx] = s.AllSettings()
		s.OnConfigChange(func(e fsnotify.Event) {
			cfgLog.Infof("changes in layer #%d: %s", idx, e.String())
			settings := s.AllSettings()
			changes <- change{idx, settings}
		})
	}

	w := new(wrapperView)
	w.cfg = merge(layers...)

	// re-merge layers upon changes in files
	go func(layers []map[string]interface{}) {
		for c := range changes {
			layers[c.idx] = c.layer
			w.delegate(merge(layers...))
		}
	}(layers)

	return w, nil
}

func merge(layers ...map[string]interface{}) *viper.Viper {
	cfg := viper.New()
	for _, m := range layers {
		cfg.MergeConfigMap(m)
	}
	return cfg
}

// Wrapper struct that implements View interface
// and delegates to other viper.Viper instance
type wrapperView struct {
	cfg *viper.Viper
	sync.Mutex
}

func (w *wrapperView) delegate(cfg *viper.Viper) {
	w.Lock()
	w.cfg = cfg
	w.Unlock()
}

func (w *wrapperView) IsSet(key string) bool {
	w.Lock()
	v := w.cfg.IsSet(key)
	w.Unlock()
	return v
}

func (w *wrapperView) GetString(key string) string {
	w.Lock()
	v := w.cfg.GetString(key)
	w.Unlock()
	return v
}

func (w *wrapperView) GetInt(key string) int {
	w.Lock()
	v := w.cfg.GetInt(key)
	w.Unlock()
	return v
}

func (w *wrapperView) GetInt64(key string) int64 {
	w.Lock()
	v := w.cfg.GetInt64(key)
	w.Unlock()
	return v
}

func (w *wrapperView) GetStringSlice(key string) []string {
	w.Lock()
	v := w.cfg.GetStringSlice(key)
	w.Unlock()
	return v
}

func (w *wrapperView) GetBool(key string) bool {
	w.Lock()
	v := w.cfg.GetBool(key)
	w.Unlock()
	return v
}

func (w *wrapperView) GetDuration(key string) time.Duration {
	w.Lock()
	v := w.cfg.GetDuration(key)
	w.Unlock()
	return v
}

func (w *wrapperView) GetStringMap(key string) map[string]interface{} {
	w.Lock()
	v := w.cfg.GetStringMap(key)
	w.Unlock()
	return v
}
