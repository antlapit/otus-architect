package event

var AllEvents = merge(
	BillingEvents,
	UserEvents,
	OrderEvents,
	ProductEvents,
)

func merge(ms ...map[string]interface{}) map[string]interface{} {
	var res = map[string]interface{}{}
	for _, m := range ms {
		for k, v := range m {
			res[k] = v
		}
	}
	return res
}
