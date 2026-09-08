# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## Вывод по итерации 17

- При снятии базового профиля, оказалось что на каждый запрос создавался и в дальнейшем выбрасывался `gzip.NewWriter(io.Writer)`, что при аллокации объектов в 25%, съедал почти всю память(99.4%).
- Было принято решение использовать `sync.Pool` для решения этой проблемы. В результате получили снижение потребления памяти(alloc_space) с 15Гб до 103Мб, но в тоже время произошло увеличение(inuse_space) из-за того, что теперь мы удерживаем в памяти writer'ы.

Результат команды:

```
go tool pprof -top -diff_base profiles/base.pprof profiles/result.pprof
```

File: server.exe

Build ID: C:\Users\user\AppData\Local\Temp\go-build2434134125\b001\exe\server.exe2026-09-04 15:32:35.7989189 +0500 +05

Type: inuse_space

Time: 2026-09-04 14:20:41 +05

Showing nodes accounting for 19074.19kB, 381.14% of 5004.48kB total

Dropped 2 nodes (cum <= 25.02kB)

| flat       | flat%   | sum%    | cum        | cum%                                                                                                           |
|------------|---------|---------|------------|----------------------------------------------------------------------------------------------------------------|
| 17149.14kB | 342.68% | 342.68% | 19331.98kB | 386.29%  compress/flate.NewWriter (inline)                                                                     |
| 2182.84kB  | 43.62%  | 386.29% | 2182.84kB  | 43.62%  compress/flate.(*compressor).initDeflate (inline)                                                      |
| 768.26kB   | 15.35%  | 401.64% | 768.26kB   | 15.35%  go.uber.org/zap/zapcore.newCounters (inline)                                                           |
| -514kB     | 10.27%  | 391.37% | -514kB     | 10.27%  bufio.NewWriterSize (inline)                                                                           |
| -512.04kB  | 10.23%  | 381.14% | -512.04kB  | 10.23%  context.withCancel (inline)                                                                            |
| 0          | 0%      | 381.14% | 2182.84kB  | 43.62%  compress/flate.(*compressor).init                                                                      |
| 0          | 0%      | 381.14% | 19331.98kB | 386.29%  compress/gzip.(*Writer).Close                                                                         |
| 0          | 0%      | 381.14% | 19331.98kB | 386.29%  compress/gzip.(*Writer).Write                                                                         |
| 0          | 0%      | 381.14% | -512.04kB  | 10.23%  context.WithCancel                                                                                     |
| 0          | 0%      | 381.14% | 19331.98kB | 386.29%  github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/gzip.(*compressWriter).Close |
| 0          | 0%      | 381.14% | 19331.98kB | 386.29%  github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/gzip.Middleware.func1        |
| 0          | 0%      | 381.14% | 768.26kB   | 15.35%  github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger.InitializeLogger       |
| 0          | 0%      | 381.14% | 19331.98kB | 386.29%  github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger.RequestLogger.func1   |
| 0          | 0%      | 381.14% | 19331.98kB | 386.29%  github.com/go-chi/chi/v5.(*Mux).Mount.func1                                                           |
| 0          | 0%      | 381.14% | 19331.98kB | 386.29%  github.com/go-chi/chi/v5.(*Mux).ServeHTTP                                                             |
| 0          | 0%      | 381.14% | 19331.98kB | 386.29%  github.com/go-chi/chi/v5.(*Mux).routeHTTP                                                             |
| 0          | 0%      | 381.14% | 768.26kB   | 15.35%  go.uber.org/zap.(*Logger).WithOptions                                                                  |
| 0          | 0%      | 381.14% | 768.26kB   | 15.35%  go.uber.org/zap.Config.Build                                                                           |
| 0          | 0%      | 381.14% | 768.26kB   | 15.35%  go.uber.org/zap.Config.buildOptions.WrapCore.func5                                                     |
| 0          | 0%      | 381.14% | 768.26kB   | 15.35%  go.uber.org/zap.Config.buildOptions.func1                                                              |
| 0          | 0%      | 381.14% | 768.26kB   | 15.35%  go.uber.org/zap.New                                                                                    |
| 0          | 0%      | 381.14% | 768.26kB   | 15.35%  go.uber.org/zap.optionFunc.apply                                                                       |
| 0          | 0%      | 381.14% | 768.26kB   | 15.35%  go.uber.org/zap/zapcore.NewSamplerWithOptions                                                          |
| 0          | 0%      | 381.14% | 768.26kB   | 15.35%  main.main                                                                                              |
| 0          | 0%      | 381.14% | 18305.94kB | 365.79%  net/http.(*conn).serve                                                                                |
| 0          | 0%      | 381.14% | 19331.98kB | 386.29%  net/http.HandlerFunc.ServeHTTP                                                                        |
| 0          | 0%      | 381.14% | -514kB     | 10.27%  net/http.newBufioWriterSize                                                                            |
| 0          | 0%      | 381.14% | 19331.98kB | 386.29%  net/http.serverHandler.ServeHTTP                                                                      |
| 0          | 0%      | 381.14% | 513kB      | 10.25%  runtime.allocm                                                                                         |
| 0          | 0%      | 381.14% | 513kB      | 10.25%  runtime.handoffp                                                                                       |
| 0          | 0%      | 381.14% | 768.26kB   | 15.35%  runtime.main                                                                                           |
| 0          | 0%      | 381.14% | -512.23kB  | 10.24%  runtime.malg                                                                                           |
| 0          | 0%      | 381.14% | 513kB      | 10.25%  runtime.mstart                                                                                         |
| 0          | 0%      | 381.14% | 513kB      | 10.25%  runtime.mstart0                                                                                        |
| 0          | 0%      | 381.14% | 513kB      | 10.25%  runtime.mstart1                                                                                        |
| 0          | 0%      | 381.14% | 513kB      | 10.25%  runtime.newm                                                                                           |
| 0          | 0%      | 381.14% | -512.23kB  | 10.24%  runtime.newproc.func1                                                                                  |
| 0          | 0%      | 381.14% | -512.23kB  | 10.24%  runtime.newproc1                                                                                       |
| 0          | 0%      | 381.14% | 513kB      | 10.25%  runtime.retake                                                                                         |
| 0          | 0%      | 381.14% | 513kB      | 10.25%  runtime.startm                                                                                         |
| 0          | 0%      | 381.14% | 513kB      | 10.25%  runtime.sysmon                                                                                         |
| 0          | 0%      | 381.14% | -512.23kB  | 10.24%  runtime.systemstack                                                                                    |