package handler

import (
	"encoding/json"
	"testing"
	"time"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/modules/commerce/service"
	"justixauto/internal/pkg/jsonx"
)

func TestReservationHTTPSnapshot(t *testing.T) {
	lot := "lot"
	requested := &service.ReservationSnapshot{Lines:[]service.ReservationSnapshotLine{{LineID:"line",SupplyLotID:&lot,ModelID:"model",ModelSpecificationVersion:"2",ExteriorColor:"white",InteriorColor:"black",Quantity:2}}}
	offered := &service.ReservationSnapshot{Lines:[]service.ReservationSnapshotLine{{LineID:"line",SupplyLotID:&lot,ModelID:"model",ModelSpecificationVersion:"2",ExteriorColor:"white",InteriorColor:"black",Quantity:3}}}
	v:=&service.ReservationView{Reservation:model.Reservation{ID:"reservation",BuyerCompanyID:"buyer",SupplierCompanyID:"supplier",Status:model.ReservationOffered,RequestedDigest:"requested-digest",OfferedDigest:"offered-digest",RequestedRevision:1,OfferedRevision:2,Version:2,CreatedAt:time.Now(),UpdatedAt:time.Now()},Requested:requested,Offered:offered,Buyer:service.Company{ID:"buyer",Name:"Dealer"},Supplier:service.Company{ID:"supplier",Name:"Distributor"},AllowedActions:[]string{"accept"}}
	raw,err:=json.Marshal(toReservation(v));if err!=nil{t.Fatal(err)}
	var dto struct { BuyerCompanyID string `json:"buyerCompanyId"`; SupplierCompanyID string `json:"supplierCompanyId"`; Requested struct{Lines []struct{Quantity jsonx.Quantity `json:"quantity"`; ModelSpecificationVersion string `json:"modelSpecificationVersion"`} `json:"lines"`} `json:"requested"`; Offered struct{Lines []struct{Quantity jsonx.Quantity `json:"quantity"`} `json:"lines"`} `json:"offered"`; Revision string `json:"revision"`; AllowedActions []string `json:"allowedActions"` }
	if err:=json.Unmarshal(raw,&dto);err!=nil{t.Fatal(err)}
	if dto.BuyerCompanyID!="buyer"||dto.SupplierCompanyID!="supplier"||dto.Requested.Lines[0].Quantity!=2||dto.Offered.Lines[0].Quantity!=3||dto.Requested.Lines[0].ModelSpecificationVersion!="2"||dto.Revision==""||len(dto.AllowedActions)!=1||dto.AllowedActions[0]!="accept"{t.Fatalf("wire=%s",raw)}
}

func TestReservationHTTPAllocationHistory(t *testing.T) {
	v:=&service.ReservationAllocationView{ReservationID:"reservation",Revision:3,Allocations:[]model.SupplyClaimAllocation{{ID:"allocation",ClaimID:"claim",ReservationLineID:"source-line",VehicleID:"vehicle",HolderType:"commerce-reservation",HolderID:"reservation",Current:false},{ID:"replacement",ClaimID:"claim",ReservationLineID:"source-line",VehicleID:"other-vehicle",HolderType:"commerce-order",HolderID:"order",Current:true}}}
	raw,err:=json.Marshal(toReservationAllocations(v));if err!=nil{t.Fatal(err)}
	var got struct{ReservationID string `json:"reservationId"`;Allocations []struct{LineID string `json:"lineId"`;VehicleID string `json:"vehicleId"`;Current bool `json:"current"`;HolderType string `json:"holderType"`} `json:"allocations"`}
	if err:=json.Unmarshal(raw,&got);err!=nil{t.Fatal(err)}
	if got.ReservationID!="reservation"||len(got.Allocations)!=2||got.Allocations[0].LineID!="source-line"||got.Allocations[0].Current||!got.Allocations[1].Current||got.Allocations[1].HolderType!="commerce-order"{t.Fatalf("wire=%s",raw)}
}
