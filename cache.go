package sample1

import (
	"time"
	"fmt"
)

// PriceService is a service that we can use to get prices for the items
// Calls to this service are expensive (they take time)
type PriceService interface {
	GetPriceFor(itemCode string) (float64, error)
}

// TransparentCache is a cache that wraps the actual service
// The cache will remember prices we ask for, so that we don't have to wait on every call
// Cache should only return a price if it is not older than "maxAge", so that we don't get stale prices
type TransparentCache struct {
	actualPriceService 		PriceService
	maxAge             		time.Duration
	prices             		map[string]Item
	maxConcurrentRoutines	int
}

type Item struct {
	price float64
	expirationTime time.Time
}


func NewTransparentCache(actualPriceService PriceService, maxAge time.Duration) *TransparentCache {
	return &TransparentCache{
		actualPriceService: actualPriceService,
		maxAge:             maxAge,
		prices:             map[string]Item{},
	}
}

// GetPriceFor gets the price for the item, either from the cache or the actual service if it was not cached or too old
func (c *TransparentCache) GetPriceFor(itemCode string) (float64, error) {

	item, ok := c.prices[itemCode]
	if ok {

		if !hasExpired(item.expirationTime,c.maxAge) {
			return item.price, nil
		}
	}

	price, err := c.actualPriceService.GetPriceFor(itemCode)
	if err != nil {
		return 0, fmt.Errorf("getting price from service : %v", err.Error())
	}

	item = Item{price,time.Now()}
	c.prices[itemCode] = item
	return price, nil
}

// GetPricesFor gets the prices for several items at once, some might be found in the cache, others might not
// If any of the operations returns an error, it should return an error as well
func (c *TransparentCache) GetPricesFor(itemCodes ...string) (results []float64,err error) {


	concurrentRoutines := make(chan int, c.maxConcurrentRoutines)

	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
			results = []float64{}
		}
	}()

	for i := 0; i < c.maxConcurrentRoutines; i++ {
		concurrentRoutines <- 1
	}

	// The done channel indicates when a single goroutine has
	// finished its job.
	done := make(chan bool)
	
	// waitForAllJobs channel allows the main program
	// to wait until we have indeed done all the calls.
	waitForAllCalls := make(chan bool)

	// Collect all the cache calls, and since the cache call is finished, we can
	// release another spot for a routine.
	go func() {
		for i := 0 ; i < len(itemCodes); i++ {
			<-done
			// Say that another goroutine can now start.
			concurrentRoutines <- 1
		}
		// We have collected all the jobs, the program
		// can now terminate
		waitForAllCalls <- true
	}()


	for _, itemCode := range itemCodes {

		<-concurrentRoutines

		go func(itemCode string) {
	
			defer func() {
				done <-true
			}()
		
			price, err := c.GetPriceFor(itemCode)
			if err != nil {
				panic(err)
			}

			results = append(results, price)
		}(itemCode)
	}

	<-waitForAllCalls

	return results, nil
}



//hasExpired checks if the item has surpassed the maxAge
func hasExpired(expirationTime time.Time,maxAge time.Duration) bool {

	currentTime := time.Now()

	resultTime := currentTime.Add(-maxAge)

	if resultTime.After(expirationTime) {
		return true
	}

	return false
}