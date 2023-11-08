# worker
go worker pool package

```golang
pool:=NewJobPool(minWorker, maxWorker, queuelen, WithWorkerPanicHandler(func(e any) {
			fmt.Printf("err:%v\n",e)
		})),
```