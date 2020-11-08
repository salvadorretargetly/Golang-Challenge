package sample1

import (
	"testing"
	"time"
	"fmt"
	"sort"
)

// mockResult has the float64 and err to return
type mockResult struct {
	price float64
	err   error
}

type mockPriceService struct {
	numCalls    int
	mockResults map[string]mockResult // what price and err to return for a particular itemCode
	callDelay   time.Duration         // how long to sleep on each call so that we can simulate calls to be expensive
}

func (m *mockPriceService) GetPriceFor(itemCode string) (float64, error) {

	m.numCalls++            // increase the number of calls
	time.Sleep(m.callDelay) // sleep to simulate expensive call

	result, ok := m.mockResults[itemCode]
	if !ok {
		panic(fmt.Errorf("bug in the tests, we didn't have a mock result for [%v]", itemCode))
	}
	return result.price, result.err
}

func (m *mockPriceService) getNumCalls() int {
	return m.numCalls
}

func getPriceWithNoErr(t *testing.T, cache *TransparentCache, itemCode string) float64 {
	price, err := cache.GetPriceFor(itemCode)
	if err != nil {
		t.Error("error getting price for", itemCode)
	}
	return price
}

func getPricesWithNoErr(t *testing.T, cache *TransparentCache, itemCodes ...string) []float64 {
	prices, err := cache.GetPricesFor(itemCodes...)
	if err != nil {
		t.Error("error getting prices for", itemCodes)
	}
	return prices
}

func assertInt(t *testing.T, expected int, actual int, msg string) {
	if expected != actual {
		t.Error(msg, fmt.Sprintf("expected : %v, got : %v", expected, actual))
	}
}

func assertFloat(t *testing.T, expected float64, actual float64, msg string) {
	if expected != actual {
		t.Error(msg, fmt.Sprintf("expected : %v, got : %v", expected, actual))
	}
}

func assertFloats(t *testing.T, expected []float64, actual []float64, msg string) {
	if len(expected) != len(actual) {
		t.Error(msg, fmt.Sprintf("expected : %v, got : %v", expected, actual))
		return
	}
	sort.Float64s(expected)
	sort.Float64s(actual)
	for i, expectedValue := range expected {
		if expectedValue != actual[i] {
			t.Error(msg, fmt.Sprintf("expected : %v, got : %v", expected, actual))
			return
		}
	}
}

// Check that we are caching results (we should not call the external service for all calls)
func TestGetPriceFor_CachesResults(t *testing.T) {
	mockService := &mockPriceService{
		mockResults: map[string]mockResult{
			"p1": {price: 5, err: nil},
			"p2": {price: 15, err: nil},
		},
	}
	cache := NewTransparentCache(mockService, time.Minute,5)
	assertFloat(t, 5, getPriceWithNoErr(t, cache, "p1"), "wrong price returned")
	assertFloat(t, 5, getPriceWithNoErr(t, cache, "p1"), "wrong price returned")
	assertFloat(t, 5, getPriceWithNoErr(t, cache, "p1"), "wrong price returned")

	assertFloat(t, 15, getPriceWithNoErr(t, cache, "p2"), "wrong price returned")
	assertFloat(t, 15, getPriceWithNoErr(t, cache, "p2"), "wrong price returned")
	assertFloat(t, 15, getPriceWithNoErr(t, cache, "p2"), "wrong price returned")

	assertInt(t, 2, mockService.getNumCalls(), "wrong number of service calls")
}

// Check that cache returns an error if external service returns an error
func TestGetPriceFor_ReturnsErrorOnServiceError(t *testing.T) {
	mockService := &mockPriceService{
		mockResults: map[string]mockResult{
			"p1": {price: 0, err: fmt.Errorf("some error")},
		},
	}
	cache := NewTransparentCache(mockService, time.Minute,5)
	_, err := cache.GetPriceFor("p1")
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

// Check that cache can return more than one price at once, caching appropriately
func TestGetPricesFor_GetsSeveralPricesAtOnceAndCachesThem(t *testing.T) {
	mockService := &mockPriceService{
		mockResults: map[string]mockResult{
			"p1": {price: 5, err: nil},
			"p2": {price: 7, err: nil},
		},
	}
	cache := NewTransparentCache(mockService, time.Minute,5)
	assertFloat(t, 5, getPriceWithNoErr(t, cache, "p1"), "wrong price returned")
	assertFloats(t, []float64{5, 7}, getPricesWithNoErr(t, cache, "p1", "p2"), "wrong price returned")
	assertFloats(t, []float64{5, 7}, getPricesWithNoErr(t, cache, "p1", "p2"), "wrong price returned")
	assertInt(t, 2, mockService.getNumCalls(), "wrong number of service calls")
}

// Check that we are expiring results when they exceed the max age
func TestGetPriceFor_DoesNotReturnOldResults(t *testing.T) {
	mockService := &mockPriceService{
		mockResults: map[string]mockResult{
			"p1": {price: 5, err: nil},
			"p2": {price: 7, err: nil},
		},
	}
	maxAge := time.Millisecond * 200
	maxAge70Pct := time.Millisecond * 140
	cache := NewTransparentCache(mockService, maxAge,5)
	// get price for "p1" twice (one external service call)
	assertFloat(t, 5, getPriceWithNoErr(t, cache, "p1"), "wrong price returned")
	assertFloat(t, 5, getPriceWithNoErr(t, cache, "p1"), "wrong price returned")
	assertInt(t, 1, mockService.getNumCalls(), "wrong number of service calls")
	// sleep 0.7 the maxAge
	time.Sleep(maxAge70Pct)
	// get price for "p1" and "p2", only "p2" should be retrieved from the external service (one more external call)
	assertFloat(t, 5, getPriceWithNoErr(t, cache, "p1"), "wrong price returned")
	assertFloat(t, 5, getPriceWithNoErr(t, cache, "p1"), "wrong price returned")
	assertFloat(t, 7, getPriceWithNoErr(t, cache, "p2"), "wrong price returned")
	assertFloat(t, 7, getPriceWithNoErr(t, cache, "p2"), "wrong price returned")
	assertInt(t, 2, mockService.getNumCalls(), "wrong number of service calls")
	// sleep 0.7 the maxAge
	time.Sleep(maxAge70Pct)
	// get price for "p1" and "p2", only "p1" should be retrieved from the cache ("p2" is still valid)
	assertFloat(t, 5, getPriceWithNoErr(t, cache, "p1"), "wrong price returned")
	assertFloat(t, 5, getPriceWithNoErr(t, cache, "p1"), "wrong price returned")
	assertFloat(t, 7, getPriceWithNoErr(t, cache, "p2"), "wrong price returned")
	assertInt(t, 3, mockService.getNumCalls(), "wrong number of service calls")
}

// Check that cache parallelize service calls when getting several values at once
func TestGetPricesFor_ParallelizeCalls(t *testing.T) {
	mockService := &mockPriceService{
		callDelay: time.Second, // each call to external service takes one full second
		mockResults: map[string]mockResult{
			"p1": {price: 5, err: nil},
			"p2": {price: 7, err: nil},
		},
	}
	cache := NewTransparentCache(mockService, time.Minute,5)
	start := time.Now()
	assertFloats(t, []float64{5, 7}, getPricesWithNoErr(t, cache, "p1", "p2"), "wrong price returned")
	elapsedTime := time.Since(start)
	if elapsedTime > (1200 * time.Millisecond) {
		t.Error("calls took too long, expected them to take a bit over one second")
	}
}

// Check that cache returns an error if one external service call fail
func TestGetPriceFor_ReturnsErrorWithMultiplesItems(t *testing.T) {
	mockService := &mockPriceService{
		mockResults: map[string]mockResult{
			"p1": {price: 5, err: nil},
			"p2": {price: 25, err: nil},
			"p3": {price: 0, err: fmt.Errorf("some error")},
		},
	}
	cache := NewTransparentCache(mockService, time.Minute,5)
	_, err := cache.GetPricesFor("p1","p2","p3")
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}



// Check that cache returns correctly passing duplicates itemCodes
func TestGetPriceFor_DuplicateItemCodes(t *testing.T) {

	mockService := &mockPriceService{
		mockResults: map[string]mockResult{
			"p1": {price: 5, err: nil},
			"p2": {price: 7, err: nil},
			"p3": {price: 3, err: nil},
		},
	}

	cache := NewTransparentCache(mockService, time.Minute,5)
	assertFloats(t, []float64{5,7,3,5,3}, getPricesWithNoErr(t, cache,"p1","p2","p3","p1","p3"), "wrong price returned")
}




// Check that cache returns  in a short time period as if we were making only one request with a big number of item codes.
func TestGetPricesFor_StressfullTest(t *testing.T) {

	mockService := &mockPriceService{
		callDelay: time.Second, // each call to external service takes one full second
		mockResults: map[string]mockResult{
			"p1": {price: 1, err: nil},
			"p2": {price: 2, err: nil},
			"p3": {price: 3, err: nil},
			"p4": {price: 4, err: nil},
			"p5": {price: 5, err: nil},
			"p6": {price: 6, err: nil},
			"p7": {price: 7, err: nil},
			"p8": {price: 8, err: nil},
			"p9": {price: 9, err: nil},
			"p10": {price: 10, err: nil},
			"p11": {price: 11, err: nil},
			"p12": {price: 12, err: nil},
			"p13": {price: 13, err: nil},
			"p14": {price: 14, err: nil},
			"p15": {price: 15, err: nil},
			"p16": {price: 16, err: nil},
		},
	}

	cache := NewTransparentCache(mockService, time.Minute,20)
	start := time.Now()
	assertFloats(t, []float64{1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16}, getPricesWithNoErr(t, cache, "p1", "p2","p3","p4","p5","p6","p7","p8","p9","p10","p11","p12","p13","p14","p15","p16"), "wrong price returned")
	elapsedTime := time.Since(start)
	if elapsedTime > (1200 * time.Millisecond) {
		t.Error("calls took too long, expected them to take a bit over one second")
	}
}