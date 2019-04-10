package config

import (
	"errors"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Reads configurations from all specified files using read(),
// and then merges them into a single viper.Viper instance.
func readMerged(files ...string) (View, error) {
	if len(files) == 0 {
		return nil, errors.New("no input files specified")
	}

	w := new(wrapperView)
	layers := make([]*viper.Viper, len(files))

	queue := make(chan fsnotify.Event, 1)
	onFileChange := func(e fsnotify.Event) {
		select {
		case queue <- e:
		default:
		}
	}

	// read files into layers and watch for changes
	for i, f := range files {
		l, err := read(f, onFileChange)
		if err != nil {
			return nil, err
		}
		layers[i] = l
	}

	w.cfg = merge(layers...)

	// re-merge layers upon changes in files
	go func() {
		for range queue {
			w.cfg = merge(layers...)
		}
	}()

	return w, nil
}

func merge(layers ...*viper.Viper) *viper.Viper {
	cfg := viper.New()
	for _, l := range layers {
		m := l.AllSettings()
		cfg.MergeConfigMap(m)
	}
	return cfg
}

// Wrapper struct that implements View interface
// and delegates to other viper.Viper instance
type wrapperView struct {
	cfg *viper.Viper
}

func (w *wrapperView) IsSet(key string) bool {
	return w.cfg.IsSet(key)
}

func (w *wrapperView) GetString(key string) string {
	return w.cfg.GetString(key)
}

func (w *wrapperView) GetInt(key string) int {
	return w.cfg.GetInt(key)
}

func (w *wrapperView) GetInt64(key string) int64 {
	return w.cfg.GetInt64(key)
}

func (w *wrapperView) GetStringSlice(key string) []string {
	return w.cfg.GetStringSlice(key)
}

func (w *wrapperView) GetBool(key string) bool {
	return w.cfg.GetBool(key)
}

func (w *wrapperView) GetDuration(key string) time.Duration {
	return w.cfg.GetDuration(key)
}

func (w *wrapperView) GetStringMap(key string) map[string]interface{} {
	return w.cfg.GetStringMap(key)
}
