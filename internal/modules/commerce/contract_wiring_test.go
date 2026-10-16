package commerce

import (
	"testing"

	"justixauto/internal/modules/commerce/repository"
	"justixauto/internal/modules/commerce/service"
)

func TestContractStoreWiring(t *testing.T) {
	a := storeAdapter{r: repository.NewStore(nil)}
	if contracts := a.Contracts(); contracts == nil {
		t.Fatal("contract store is not wired")
	}
	var _ service.Store = a
}
