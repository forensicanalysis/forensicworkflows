package daggy

// Data is the input into the plugins.
type Data map[string]interface{}

// Plugin is an interface that all plugins need to implement.
type Plugin interface {
	Run(store string, data Data) error
}
