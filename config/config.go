package config

type Config interface {
	Get(key string) interface{}
	Set(key string, data interface{})
	String(key string) string
	Int(key string) int
	Float(key string) float64
}
