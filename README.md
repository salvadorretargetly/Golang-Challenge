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