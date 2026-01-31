## 01/15/2026 
Direct call to openLibraryAPI is complete. This project is now just a wrapper for the openLibraryAPI call.

## 01/20/2026
Redis Cache is implemented so now if a user has a repeated search, it will hit redis cace first --> also reduces the time it takes to spit out a result (milliseconds to microseconds)

Need to look into making changes to support fuzzy searches.

Ok nvrm I added fuzzy searches, but upon testing there are some cases that are not passing:
for example:
- when searching project+hail+mary, it is apperantly different from project+hail+maryyy I need to figure out a better algo for search...

## 01/31/2026
Ok it's been a while since I checked in on this jawn... I'm looking into why the search algo is giving different results from "project+hail+maryyy" vs "project+hail+mary". Probably has something to do with string trimming or the L-distance not accounting for repeating end characters... I'll check into it tomorrow...
