## 01/15/2026 
Direct call to openLibraryAPI is complete. This project is now just a wrapper for the openLibraryAPI call.

## 01/20/2026
Redis Cache is implemented so now if a user has a repeated search, it will hit redis cace first --> also reduces the time it takes to spit out a result (milliseconds to microseconds)

Need to look into making changes to support fuzzy searches.