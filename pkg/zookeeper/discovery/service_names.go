package discovery

import (
	"strings"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"golang.org/x/exp/slices"
)

type NameAdaptor struct {
	C2KNamesCache *expirable.LRU[string, string]
	K2CNamesCache *expirable.LRU[string, string]
}

func (op *NameAdaptor) KtoCName(kName string) string {
	if cName, exists := op.K2CNamesCache.Get(kName); exists {
		op.K2CNamesCache.Add(kName, cName)
		op.C2KNamesCache.Add(cName, kName)
		return cName
	}
	return ""
}

func (op *NameAdaptor) CToKName(cName string) string {
	if kName, exists := op.C2KNamesCache.Get(cName); exists {
		op.C2KNamesCache.Add(cName, kName)
		op.K2CNamesCache.Add(kName, cName)
		return kName
	}
	tName := strings.ToLower(cName)
	tNameSegs := strings.Split(tName, `.`)
	slices.Reverse(tNameSegs)
	knameLen := 0
	knameIdx := len(tNameSegs)
	for idx, tNameSeg := range tNameSegs {
		len := knameLen + 1 + len(tNameSeg)
		if len >= 64 {
			knameIdx = idx
			break
		}
		knameLen = len
	}
	kName := strings.Join(tNameSegs[0:knameIdx], "-")
	op.C2KNamesCache.Add(cName, kName)
	op.K2CNamesCache.Add(kName, cName)
	return kName
}
