# Tracking down a Memory Leak in Go/SQLite

run `make test` - WARNING: long running - several minutes on my workstation

---

OSs supported:
* Windows_NT => memory leak not observable - _at least not to the same extent_
  * uses `tasklist` to measure process memory footprint
* Linux => ***memory leak observed***
  * uses `ps` to measure process memory footprint

---

First, the test starts a OS child process that
* contains a "large" in-mem sqlite db filled with dummy data
* offers a web endpoint `http://localhost:8890/dumpdb` ... once the db is initialized

Then, the test loops over
* calling that web endpoint of its testee, who
  * makes an in-mem copy - I call it a "snapshot" - of the db using SQLite's backup mechanism copying memory pages
  * dumps that snapshot as INSERT stmts into a file using `github.com/schollz/sqlite3dump`
    
    *note: this is a fairly long operation, that blocks the db, which is why we are using that "snapshot" mechanism*
  * and [supposedly] disposes that snapshot
* records memory usage (rss) of the testee using OS tools

At the end, the recorded memory usage is printed for each loop.

---

example results (process memory footprint - in KB) on my work station:

| Iteration  |  Win |     Lin | Lin with WORKAROUND<br/>(file based snapshot db) |
|:----------:|-----:|--------:|-------------------------------------------------:|
|     0      | 625’928 |  592132 |                                             593480 |
|     1      | 637’648 | 1005364 |                                             595088 |
|     2      | 637’188 | 1429180 |                                             594708 |
|     3      | 637’396 | 1429928 |                                             594408 |
|     4      | 638’196 | 1585408 |                                             594280 |
|     5      | 636’824 | 1869536 |                                             594568 |
|     6      | 638’256 | 2347204 |                                             594480 |
|     7      | 637’596 | 2757300 |                                             594864 |
|     8      | 638’728 | 2822888 |                                             594816 |
|     9      | 637’732 | 2822320 |                                             597340 |
|     10     | 648’708 | 3280812 |                                             597196 |
|     11     | 671’804 | 3609528 |                                             597048 |
|     12     | 672’932 | 4133240 |                                             598112 |
|     13     | 718’468 | 4264400 |                                             597220 |
|     14     | 834’328 | 4788828 |                                             597144 |
|     15     | 832’080 | 4854132 |                                             597308 |
|     16     | 845’052 | 4854172 |                                             598084 |
|     17     | 832’348 | 4854140 |                                             597512 |
|     18     | 836’280 | 4914684 |                                             597648 |
|     19     | 838’816 | 5328012 |                                             597424 |
|     20     | 844’752 | 5392820 |                                             598224 |



### Some observations
* Not in every iteration of every run a memory growth can be observed, but running our productive online
  service 24/7, we can monitor continuous growth over time ***always leading to a OutOfMemory exception***.
  
* It seems to be a ***linux related problem***. 

  At least this test shows not the same memory growth when run in Windows.

* We assume the ***in-mem*** "snapshot" of an ***in-mem*** db to be a part of the problem.

  Why? Our current ***workaround*** (search for code comment with `WORKAROUND`) is to change the "snapshot" db to be a file based db by patching `mode` in the 
  connection string of the snapshot db: `memory` -> `rwc`, which makes the memory growth disappear.

  Why "part of"? Also, the fact of having snapshot db activity (in our case: db dump - search for code comment with `snapshot db activity``) seems to affect the memory behavior.
  Without snapshot db activity - just change the corresponding code line - the growth seems to be capped after ~6 
  iterations. 

* the ***schema complexity*** of the db seems to influence the problem.
  
  When [further] simplifying the db schema, the problem seems to disappear even when adjusting the db size of the 
  remaining tables by adding more rows. That is why we left the structure as shown in this repo.

* As the ***go managed memory is not increasing***, but only the OS view process footprint, we suspect sqlite
  to be the culprit.

=> _our assumption: the snapshot db is not properly "disposed"._
_What's wrong with our code? Or is it a problem somewhere in golang or even deeper in SQLite code?_
