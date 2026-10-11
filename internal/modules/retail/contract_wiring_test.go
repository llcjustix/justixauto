package retail

import (
	"testing"

	"justixauto/internal/modules/retail/repository"
	"justixauto/internal/modules/retail/service"
)

func TestContractStoreWiring(t *testing.T) {
	var _ service.Store = storeAdapter{}
	if (storeAdapter{r: repository.NewStore(nil)}).Contracts() == nil {
		t.Fatal("contract repository adapter is absent")
	}
}
