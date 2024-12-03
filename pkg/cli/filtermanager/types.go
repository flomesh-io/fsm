package filtermanager

import "sigs.k8s.io/gwctl/pkg/common"

type FilterManager struct {
	Fetcher common.GroupKindFetcher
}
