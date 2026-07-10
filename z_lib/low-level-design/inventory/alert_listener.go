package inventory

type AlertListener interface {
	OnLowStock(warehouseID string, productID string, currentQuantity int)
}