# Golang-Challenge

Step by Step

1- First we create an item struct with the properties price and expirationTime. The last property is necessary to know the last time the item was
saved in the cache. 

2- We add a new function hasExpired to check if the item expirationTime has surpass the maxAge.


3- We add a new property inside the cache struct to manage the concurrence to get the price of a list of items. This propertie allows you set a limit of concurrent 
calls to not overhead the price's service. For example if you set maxConcurrentRoutines = 5 the service wil always manage 5 routines trying to get the price at the same time.


4- We added the goroutines logic inside the function GetPricesFor to manage several calls at the same time.


5- We maded some changes in test module. We added the new maxConcurrentRoutines param for the cache's constructor. At the same time we verify that all test pass successfully.
We had to add a mutex to solve an issue with the concurrent write to the price mapping. All the test pass successfully.

6- We make some refactor of the handling response for the cache go routine in fuction GetPricesFor. Apart from that we add logic to make sure return the prices in the correct order, that means 
for example that if you send by param a list of item codes in an specific order you must receive their correspond prices in the same order:

Example: p2,p15,p18,p1,p2 => price p2, price p15, price p18, price p1, price p2

We verify if there were sent duplicates item codes by param and in these case we ask to the cache only the uniques. This is because we dont want to make unnecessary calls to the cache.


7- We add new unit test.
    -TestGetPriceFor_ReturnsErrorWithMultiplesItems: Check that cache returns an error if one external service call fail
    -TestGetPriceFor_DuplicateItemCodes: Check that cache returns correctly passing duplicates itemCodes
    -TestGetPricesFor_StressfulTest: Check that cache returns  in a short time period as if we were making only one request with a big number of item codes.

