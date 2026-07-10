package inventory

import (
	"errors"
	"sync"
)

type alertToFire struct {
	listener AlertListener
	productID string
	quantity int
}

type Warehouse struct {
	id string
	inventory map[string]int
	alertConfigs map[string][]*AlertConfig
	mu sync.Mutex
}

func NewWarehouse(id string) *Warehouse {
	return &Warehouse{
		id: id,
		inventory: make(map[string]int),
		alertConfigs: make(map[string][]*AlertConfig),
	}
}

func (w *Warehouse) GetID() string {
	return w.id
}

func (w *Warehouse) AddStock(productID string, quantity int) error {
	if quantity <= 0 {
		return errors.New("quantity must be positive")
	}
	var alertsToFire []alertToFire
	w.mu.Lock()
	currentQty := w.inventory[productID]
	newQty := currentQty + quantity
	w.inventory[productID] = newQty
	alertsToFire = w.getAlertsToFire(productID, currentQty, newQty)
	w.mu.Unlock()

	w.fireAlerts(alertsToFire)
	return nil
}


func (w *Warehouse) RemoveStock(productID string, quantity int) bool {
	if quantity <= 0 {
		return false
	}
	var alertsToFire []alertToFire

	w.mu.Lock()
	currentQty := w.inventory[productID]
	if currentQty < quantity {
		w.mu.Unlock()
		return false
	}

	newQty := currentQty - quantity
	w.inventory[productID] = newQty
	alertsToFire = w.getAlertsToFire(productID, currentQty, newQty)
	w.mu.Unlock()

	w.fireAlerts(alertsToFire)
	return true
}

func (w *Warehouse) GetStock(productID string) int {
	w.mu.Lock()
	defer w.mu.Lock()
	return w.inventory[productID]
}

func (w *Warehouse) CheckAvailability(productID string, quantity int) bool {
	if quantity <= 0 {
		return false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.inventory[productID] >= quantity
}

func (w *Warehouse) SetLowStockAlert(productID string, threshold int, listener AlertListener) error {
	if threshold <= 0 {
		return errors.New("threshold must be positive")
	}
	if listener == nil {
		return errors.New("listener cannot be nil")
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	config := NewAlertConfig(threshold, listener)
	w.alertConfigs[productID] = append(w.alertConfigs[productID], config)
	return nil
}

// Internal method for atomic transfers - caller must hold the mutex
func (w *Warehouse) GetStockInternal(productID string) int {
	return w.inventory[productID]
}

func (w *Warehouse) SetStockInternal(productID string, quantity int) {
	w.inventory[productID] = quantity
}

// RemoveStockInternal removes stock and return alerts to fire. Caller must hold the mutex.
func (w *Warehouse) RemoveStockInternal(productID string, quantity int) (bool, []alertToFire) {
	currentQty := w.inventory[productID]
	if currentQty < quantity {
		return false, nil
	}
	newQty := currentQty - quantity
	w.inventory[productID] = newQty
	return true, w.getAlertsToFire(productID, currentQty, newQty)
}

// AddStockInternal adds stocks and returns alerts to fire. Caller must hold a mutex to fire
func (w *Warehouse) AddStockInternal(productID string, quantity int) []alertToFire {
	currentQty := w.inventory[productID]
	newQty := currentQty + quantity
	w.inventory[productID] = newQty
	return w.getAlertsToFire(productID, currentQty, newQty)
}

func (w *Warehouse) Lock() {
	w.mu.Lock()
}

func (w *Warehouse) Unlock() {
	w.mu.Unlock()
}

func (w *Warehouse) getAlertsToFire(productID string, previousQty int, newQty int) []alertToFire {
	configs, ok := w.alertConfigs[productID]
	if !ok {
		return nil
	}

	var alertsToFire []alertToFire
	for _, config := range configs {
		if previousQty >= config.GetThreshold() && newQty < config.GetThreshold() {
			alertsToFire = append(alertsToFire, alertToFire{
				listener: config.GetListener(),
				productID: productID,
				quantity: newQty,
			})
		}
	}
	return alertsToFire
}


func (w *Warehouse) fireAlerts(alerts []alertToFire) {
	for _,alert := range alerts {
		alert.listener.OnLowStock(w.id, alert.productID, alert.quantity)
	}
}

func (w *Warehouse) FireAlerts(alerts []alertToFire) {
	w.fireAlerts(alerts)
}