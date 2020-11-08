package sample1

import (
	"time"
	"fmt"
	"sync"
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
	pricesMutex 			*sync.RWMutex
}

type Item struct {
	price float64
	expirationTime time.Time
}

type Result struct {
	itemCode string
	price	float64
	err	error
}


func NewTransparentCache(actualPriceService PriceService, maxAge time.Duration,maxConcurrentRoutines int) *TransparentCache {
	return &TransparentCache{
		actualPriceService: actualPriceService,
		maxAge:             maxAge,
		prices:             map[string]Item{},
		maxConcurrentRoutines:	maxConcurrentRoutines,
		pricesMutex:			&sync.RWMutex{},
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
	c.pricesMutex.Lock()
	c.prices[itemCode] = item
	c.pricesMutex.Unlock()

	return price, nil
}

// GetPricesFor gets the prices for several items at once, some might be found in the cache, others might not
// If any of the operations returns an error, it should return an error as well
func (c *TransparentCache) GetPricesFor(itemCodes ...string) (results []float64,err error) {

	concurrentRoutines := make(chan int, c.maxConcurrentRoutines)

	results = make([]float64,len(itemCodes))

	// The done channel indicates when a single goroutine has
	// finished its job.
	done := make(chan bool)

	resultChannel := make(chan Result,len(itemCodes))
	itemCodesChannel := make(chan string,len(itemCodes))

	//This map will save the itemCode index position
	//We we must make sure return the prices in the same order , same place as the item's code in the function param because of the concurrence
	// Example: [p1,p2,p3,p4] => [price p1, price p2, price p3, price p4]
	itemCodePositionMapping := make(map[string][]int,0)


	//We load the itemCodePositionMapping and the unique item's codes in the channel
	for position ,itemCode := range itemCodes {

		if _, ok := itemCodePositionMapping[itemCode] ; !ok  {
			itemCodePositionMapping[itemCode] = make([]int,0)
			itemCodesChannel<-itemCode
		}

		itemCodePositionMapping[itemCode] = append(itemCodePositionMapping[itemCode],position)
	}


	uniqueItemCodesLen := len(itemCodesChannel)

	for i := 0; i < c.maxConcurrentRoutines; i++ {
		concurrentRoutines <- 1
	}

	// waitForAllCalls channel allows the main program
	// to wait until we have indeed done all the calls.
	waitForAllCalls := make(chan error)

	// Collect all the cache calls, and since the cache call is finished, we can
	// release another spot for a routine.
	go func() {
		for i := 0 ; i < uniqueItemCodesLen ; i++ {
			
			<-done

			result := <- resultChannel

			if result.err != nil {
				
				//If an error ocurred we close the channels to finish inmediately with the routines execution
				close(concurrentRoutines)
				close(itemCodesChannel)
				waitForAllCalls <- result.err
				return 
		
			} else {				

				//This function make sure save the item code price in the correspondent position.
				saveItemPriceInTheCorrectPosition(&results,&itemCodePositionMapping,result.itemCode,result.price)
		
			}

			// Say that another goroutine can now start.
			concurrentRoutines <- 1
		}

		// We have collected all the calls, the program
		// can now terminate
		close(concurrentRoutines)
		close(itemCodesChannel)
		waitForAllCalls <- nil
	}()

	//At this point the itemCodesChannel has only uniques codes. This is because we dont want to make more than once request for the same code.
	for itemCode := range itemCodesChannel {

		<-concurrentRoutines

		go func(itemCode string) {
	
			defer func() {
				done <-true
			}()
		
			price, err := c.GetPriceFor(itemCode)
			
			if err != nil {
				resultChannel <- Result{itemCode,0,err}				
				return
			}

			resultChannel <- Result{itemCode,price,nil}				
			return

		}(itemCode)

	}

	err = <-waitForAllCalls

	if err != nil {
		return []float64{},err
	}

	return results, nil
}


func saveItemPriceInTheCorrectPosition(results *[]float64,itemCodePositionMapping *map[string][]int,itemCode string,price float64) {

	positions := (*itemCodePositionMapping)[itemCode]

	for _, index := range positions {
		(*results)[index] = price
	}
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