package inventory

import (
	"errors"
	"strings"
)

type InventoryManager struct {
	warehouses map[string]*Warehouse
}

func NewInventoryManager(warehouseIDs []string) *InventoryManager {
	warehouses := make(map[string]*Warehouse)
	for _, id := range warehouseIDs {
		warehouses[id] = NewWarehouse(id)
	}
	return &InventoryManager{warehouses: warehouses}
}

func (m *InventoryManager) AddStock(warehouseID string, productID string, quantity int) error {
	warehouse, ok := m.warehouses[warehouseID]
	if !ok {
		return errors.New("warehouse not found: " + warehouseID)
	}
	return warehouse.AddStock(productID, quantity)
}

func (m *InventoryManager) RemoveStock(warehouseID string, productID string, quantity int) bool {
	warehouse, ok := m.warehouses[warehouseID]
	if !ok {
		return false
	}
	return warehouse.RemoveStock(productID, quantity)
}

func (m *InventoryManager) Transfer(productID string, fromWarehouseID string, toWarehouseID string, quantity int) bool {
	if quantity <= 0 {
		return false
	}
	if fromWarehouseID == toWarehouseID {
		return false
	}
	fromWarehouse, ok := m.warehouses[fromWarehouseID]
	if !ok {
		return false
	}
	toWarehouse, ok := m.warehouses[toWarehouseID]
	if !ok {
		return false
	}
	// Lock in consistent order to prevent deadlock
	var firstWarehouse, secondWarehouse *Warehouse
	if strings.Compare(fromWarehouseID, toWarehouseID) < 0 {
		firstWarehouse = fromWarehouse
		secondWarehouse = toWarehouse
	} else {
		firstWarehouse = toWarehouse
		secondWarehouse = fromWarehouse
	}
	var fromAlerts, toAlerts []alertToFire
	var success bool

	firstWarehouse.Lock()
	secondWarehouse.Lock()

	success, fromAlerts = fromWarehouse.RemoveStockInternal(productID, quantity)
	if !success {
		secondWarehouse.Unlock()
		firstWarehouse.Unlock()
		return false
	}

	toAlerts = toWarehouse.AddStockInternal(productID, quantity)
	secondWarehouse.Unlock()
	firstWarehouse.Unlock()

	// Fire alerts outside the locks
	fromWarehouse.FireAlerts(fromAlerts)
	toWarehouse.FireAlerts(toAlerts)
	return true
}


func (m *InventoryManager) GetWarehousesWithAvailability(productID string, quantity int) []string {
	var available []string
	for warehouseID, warehouse := range m.warehouses {
		if warehouse.CheckAvailability(productID, quantity) {
			available = append(available, warehouseID)
		}
	}
	return available
}




func (m *InventoryManager) SetLowStockAlert(warehouseID string, productID string, threshold int, listener AlertListener) error {
	warehouse, ok := m.warehouses[warehouseID]
	if !ok { 
		return errors.New("warehouse not found: " + warehouseID)
	}
	return warehouse.SetLowStockAlert(productID, threshold, listener)
}