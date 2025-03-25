package reconciler

import "fmt"

var (
	errSyncingCaches = fmt.Errorf("failed initial cache sync for reconciler informers")
	errInitInformers = fmt.Errorf("informer not initialized")
)
