package gpu

var RegistryList = []GPUInfoLoader{}

func Register(loader GPUInfoLoader) {
	RegistryList = append(RegistryList, loader)
}

func GetAllGPULoaders() []GPUInfoLoader {
	return RegistryList
}
