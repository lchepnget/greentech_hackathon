package actions

import (
	"encoding/json"
	"fmt"
	"github.com/gobuffalo/buffalo"
	"net/http"
	"strings"
	"sync"
	"time"
)

var frontStore = struct {
	sync.RWMutex
	listings, orders, invoices []map[string]interface{}
}{}

func frontJSON(c buffalo.Context, v interface{}) error { return c.Render(http.StatusOK, r.JSON(v)) }
func FrontendListings(c buffalo.Context) error {
	frontStore.RLock()
	defer frontStore.RUnlock()
	return frontJSON(c, map[string]interface{}{"items": frontStore.listings, "page": 1, "pages": 1, "total": len(frontStore.listings)})
}
func FrontendListingSummary(c buffalo.Context) error {
	frontStore.RLock()
	defer frontStore.RUnlock()
	return frontJSON(c, map[string]interface{}{"totalSatsMoved": 0, "newListings": len(frontStore.listings), "regions": []string{}})
}
func FrontendListingByID(c buffalo.Context) error {
	id := c.Param("id")
	frontStore.RLock()
	defer frontStore.RUnlock()
	for _, x := range frontStore.listings {
		if x["id"] == id {
			return frontJSON(c, x)
		}
	}
	return c.Render(http.StatusNotFound, r.JSON(map[string]string{"error": "listing not found"}))
}
func FrontendCreateListing(c buffalo.Context) error {
	x := map[string]interface{}{}
	if strings.HasPrefix(c.Request().Header.Get("Content-Type"), "multipart/form-data") {
		if err := c.Request().ParseMultipartForm(10 << 20); err != nil {
			return c.Render(400, r.JSON(map[string]string{"error": "invalid multipart payload"}))
		}
		for _, key := range []string{"title", "wasteType", "quantity", "unit", "priceSats", "location", "description"} {
			x[key] = c.Request().FormValue(key)
		}
	} else if err := json.NewDecoder(c.Request().Body).Decode(&x); err != nil {
		return c.Render(400, r.JSON(map[string]string{"error": "invalid payload"}))
	}
	x["id"] = fmt.Sprintf("listing_%d", time.Now().UnixNano())
	frontStore.Lock()
	frontStore.listings = append(frontStore.listings, x)
	frontStore.Unlock()
	return c.Render(http.StatusCreated, r.JSON(x))
}
func FrontendOrders(c buffalo.Context) error {
	frontStore.RLock()
	defer frontStore.RUnlock()
	return frontJSON(c, frontStore.orders)
}
func FrontendCreateOrder(c buffalo.Context) error {
	var q struct {
		ListingID string `json:"listingId"`
		Quantity  int    `json:"quantity"`
	}
	_ = c.Bind(&q)
	if q.Quantity < 1 {
		q.Quantity = 1
	}
	x := map[string]interface{}{"id": fmt.Sprintf("order_%d", time.Now().UnixNano()), "listingId": q.ListingID, "quantity": q.Quantity, "amountSats": q.Quantity * 1000, "status": "pending", "createdAt": time.Now().UTC().Format(time.RFC3339)}
	frontStore.Lock()
	frontStore.orders = append(frontStore.orders, x)
	frontStore.Unlock()
	return c.Render(http.StatusCreated, r.JSON(x))
}
func FrontendCreateInvoice(c buffalo.Context) error {
	var q map[string]interface{}
	_ = c.Bind(&q)
	id := fmt.Sprintf("inv_%d", time.Now().UnixNano())
	x := map[string]interface{}{"id": id, "orderId": q["orderId"], "amountSats": q["amountSats"], "bolt11": "lnbc1placeholder", "status": "pending"}
	frontStore.Lock()
	frontStore.invoices = append(frontStore.invoices, x)
	frontStore.Unlock()
	return c.Render(http.StatusCreated, r.JSON(x))
}
func FrontendInvoiceStatus(c buffalo.Context) error {
	id := c.Param("id")
	frontStore.RLock()
	defer frontStore.RUnlock()
	for _, x := range frontStore.invoices {
		if x["id"] == id {
			return frontJSON(c, x)
		}
	}
	return c.Render(404, r.JSON(map[string]string{"error": "invoice not found"}))
}
func FrontendWallet(c buffalo.Context) error {
	return frontJSON(c, map[string]interface{}{"balanceSats": 0})
}
func FrontendWalletTransactions(c buffalo.Context) error { return frontJSON(c, []interface{}{}) }
func FrontendWalletDeposit(c buffalo.Context) error      { return FrontendCreateInvoice(c) }
func FrontendWalletWithdraw(c buffalo.Context) error {
	return frontJSON(c, map[string]interface{}{"id": fmt.Sprintf("tx_%d", time.Now().UnixNano()), "type": "withdraw", "status": "pending", "amountSats": 0})
}
func FrontendUpdateUser(c buffalo.Context) error {
	var x map[string]interface{}
	_ = c.Bind(&x)
	return frontJSON(c, x)
}
func FrontendForgotPassword(c buffalo.Context) error { return c.Render(http.StatusNoContent, nil) }
func FrontendResetPassword(c buffalo.Context) error  { return c.Render(http.StatusNoContent, nil) }
