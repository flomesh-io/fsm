package cache

// GatewayClassesTrigger is responsible for processing GatewayClass objects
type GatewayClassesTrigger struct{}

// Insert adds the GatewayClass object to the cache and returns true if the cache was modified
func (p *GatewayClassesTrigger) Insert(obj interface{}, cache *GatewayCache) bool {
	//class, ok := obj.(*gwv1.GatewayClass)
	//if !ok {
	//	log.Error().Msgf("unexpected object type %T", obj)
	//	return false
	//}

	//key := class.GetName()

	//class, err := cache.informers.GetListers().GatewayClass.Get(key)
	//if err != nil {
	//	log.Error().Msgf("Failed to get GatewayClass %s: %s", key, err)
	//	return false
	//}

	//if utils.IsEffectiveGatewayClass(class) {
	//	cache.mutex.Lock()
	//	defer cache.mutex.Unlock()
	//
	//	cache.gatewayclass = class
	//	return true
	//}
	//
	//return false

	return true
}

// Delete removes the GatewayClass object from the cache and returns true if the cache was modified
func (p *GatewayClassesTrigger) Delete(obj interface{}, cache *GatewayCache) bool {
	//class, ok := obj.(*gwv1.GatewayClass)
	//if !ok {
	//	log.Error().Msgf("unexpected object type %T", obj)
	//	return false
	//}

	//cache.mutex.Lock()
	//defer cache.mutex.Unlock()
	//
	//if cache.gatewayclass == nil {
	//	return false
	//}
	//
	//if class.Name == cache.gatewayclass.Name {
	//	cache.gatewayclass = nil
	//	return true
	//}
	//
	//return false

	return true
}
