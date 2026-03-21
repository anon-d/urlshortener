# Профилирование памяти (pprof)
- `base.pprof` — до оптимизации
- `result.pprof` — после оптимизации

# Оптимизации
- Кэш: `map[string]any` → `map[string]string` (убран boxing при каждой записи)
- `sync.Mutex` → `sync.RWMutex` в кэше (параллельные чтения не блокируют друг друга)
- `ShortenURL`: сигнатура `[]byte` → `string` (убраны лишние конвертации)
- `GenerateID`: `sync.Pool` для переиспользования буфера `[]byte`
- Middleware: `sync.Pool` для `gzip.Writer`

# Результат сравнения профилей

```bash
$ go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof

File: service.test
Type: inuse_space
Showing nodes accounting for 1539kB, 300.00% of 513kB total
      flat  flat%   sum%        cum   cum%
    1539kB 300.00% 300.00%     1539kB 300.00%  runtime.mallocgc
         0     0% 300.00%     1539kB 300.00%  runtime.allocm
         0     0% 300.00%     1026kB 200.00%  runtime.mcall
         0     0% 300.00%      513kB   100%  runtime.mstart
         0     0% 300.00%      513kB   100%  runtime.mstart0
         0     0% 300.00%      513kB   100%  runtime.mstart1
         0     0% 300.00%     1539kB 300.00%  runtime.newm
         0     0% 300.00%     1539kB 300.00%  runtime.newobject
         0     0% 300.00%     1026kB 200.00%  runtime.park_m
         0     0% 300.00%     1026kB 200.00%  runtime.resetspinning
         0     0% 300.00%     1539kB 300.00%  runtime.schedule
         0     0% 300.00%     1539kB 300.00%  runtime.startm
         0     0% 300.00%     1539kB 300.00%  runtime.wakep
```

# Description

`result.pprof`
```bash
File: service.test
Type: alloc_space
Time: 2026-03-21 22:55:56 MSK
Showing nodes accounting for 3615.50kB, 100% of 3615.50kB total
      flat  flat%   sum%        cum   cum%
 1053.31kB 29.13% 29.13%  1053.31kB 29.13%  github.com/anon-d/urlshortener/internal/service.(*mockProfileCacheService).Set
  513.12kB 14.19% 43.33%  1557.39kB 43.08%  github.com/anon-d/urlshortener/internal/service.(*Service).ShortenBatchURL
     513kB 14.19% 57.51%      513kB 14.19%  runtime.mallocgc
  512.04kB 14.16% 71.68%  1033.09kB 28.57%  github.com/anon-d/urlshortener/internal/service.(*Service).ShortenURL
  512.02kB 14.16% 85.84%   512.02kB 14.16%  fmt.Sprintf
  512.01kB 14.16%   100%   512.01kB 14.16%  encoding/base64.(*Encoding).EncodeToString
         0     0%   100%   512.01kB 14.16%  github.com/anon-d/urlshortener/internal/service.GenerateID
         0     0%   100%  3102.50kB 85.81%  github.com/anon-d/urlshortener/internal/service.TestProfileMemory
         0     0%   100%      513kB 14.19%  runtime.allocm
         0     0%   100%      513kB 14.19%  runtime.mstart
         0     0%   100%      513kB 14.19%  runtime.mstart0
         0     0%   100%      513kB 14.19%  runtime.mstart1
         0     0%   100%      513kB 14.19%  runtime.newm
         0     0%   100%      513kB 14.19%  runtime.newobject
         0     0%   100%      513kB 14.19%  runtime.resetspinning
         0     0%   100%      513kB 14.19%  runtime.schedule
         0     0%   100%      513kB 14.19%  runtime.startm
         0     0%   100%      513kB 14.19%  runtime.wakep
         0     0%   100%  3102.50kB 85.81%  testing.tRunner
```
         
`base.pprof`
```bash
File: service.test
Type: alloc_space
Time: 2026-03-21 13:56:08 MSK
Showing nodes accounting for 9857.91kB, 100% of 9857.91kB total
      flat  flat%   sum%        cum   cum%
 3193.56kB 32.40% 32.40%  3193.56kB 32.40%  github.com/anon-d/urlshortener/internal/service.(*mockProfileCacheService).Set
    2052kB 20.82% 53.21%     2052kB 20.82%  runtime.mallocgc
 2048.05kB 20.78% 73.99%  2048.05kB 20.78%  fmt.Sprintf
 1027.13kB 10.42% 84.41%  3156.17kB 32.02%  github.com/anon-d/urlshortener/internal/service.(*Service).ShortenBatchURL
  513.12kB  5.21% 89.61%  7805.91kB 79.18%  github.com/anon-d/urlshortener/internal/service.TestProfileMemory
  512.04kB  5.19% 94.81%  2088.57kB 21.19%  github.com/anon-d/urlshortener/internal/service.(*Service).ShortenURL
  512.01kB  5.19%   100%   512.01kB  5.19%  encoding/base64.(*Encoding).EncodeToString
         0     0%   100%   512.01kB  5.19%  github.com/anon-d/urlshortener/internal/service.GenerateID
         0     0%   100%     2052kB 20.82%  runtime.allocm
         0     0%   100%     1026kB 10.41%  runtime.mcall
         0     0%   100%     1026kB 10.41%  runtime.mstart
         0     0%   100%     1026kB 10.41%  runtime.mstart0
         0     0%   100%     1026kB 10.41%  runtime.mstart1
         0     0%   100%     2052kB 20.82%  runtime.newm
         0     0%   100%     2052kB 20.82%  runtime.newobject
         0     0%   100%     1026kB 10.41%  runtime.park_m
         0     0%   100%     1539kB 15.61%  runtime.resetspinning
         0     0%   100%     2052kB 20.82%  runtime.schedule
         0     0%   100%     2052kB 20.82%  runtime.startm
         0     0%   100%     2052kB 20.82%  runtime.wakep
         0     0%   100%  7805.91kB 79.18%  testing.tRunner
```
