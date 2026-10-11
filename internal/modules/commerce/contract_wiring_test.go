package commerce

import (
	"justixauto/internal/modules/commerce/repository"
	"justixauto/internal/modules/commerce/service"
	"testing"
)

func TestContractStoreWiring(t *testing.T) {
	a := storeAdapter{r: repository.NewStore(nil)}
	var _ service.ContractRepository = a.Contracts()
	var _ service.Store = a
}
