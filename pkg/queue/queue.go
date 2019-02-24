package queue

var (
	// Loggerf can be used to override the default logging destination. Such
	// log messages in this library should be logged at info or higher.
	Loggerf = func(format string, args ...interface{}) {}
)

type Job interface {
	// for logging or stats
	GetName() string

	// Fire the job
	Fire() error
}

type worker struct {
	id         int
	jobQueue   chan Job
	workerPool chan chan Job
	quitChan   chan bool
}

func newWorker(id int, workerPool chan chan Job) *worker {
	return &worker{
		id:         id,
		jobQueue:   make(chan Job),
		workerPool: workerPool,
		quitChan:   make(chan bool),
	}
}

func (w *worker) start() {
	go func() {
		for {
			w.workerPool <- w.jobQueue

			select {
			case job := <-w.jobQueue:
				Loggerf("worker%d: started %s job", w.id, job.GetName())
				err := job.Fire()
				if err != nil {
					Loggerf("worker%d: %s job failed with %s", w.id, job.GetName(), err.Error())
				} else {
					Loggerf("worker%d: %s job completed", w.id, job.GetName())
				}

			case <-w.quitChan:
				Loggerf("worker%d quitting", w.id)
			}
		}
	}()
}

func (w *worker) stop() {
	go func() {
		w.quitChan <- true
	}()
}

type Dispatcher struct {
	workerPool chan chan Job
	maxWorkers int
	jobQueue   chan Job
}

func NewDispatcher(jobQueue chan Job, maxWorkers int) *Dispatcher {
	workerPool := make(chan chan Job, maxWorkers)

	return &Dispatcher{
		jobQueue:   jobQueue,
		maxWorkers: maxWorkers,
		workerPool: workerPool,
	}
}

func (d *Dispatcher) Run() {
	for i := 0; i < d.maxWorkers; i++ {
		w := newWorker(i+1, d.workerPool)
		w.start()
	}

	go d.dispatch()
}

func (d *Dispatcher) dispatch() {
	for {
		select {
		case job := <-d.jobQueue:
			go func() {
				Loggerf("fetching workerJobQueue for: %s\n", job.GetName())
				workerJobQueue := <-d.workerPool
				Loggerf("adding %s to workerJobQueue\n", job.GetName())
				workerJobQueue <- job
			}()
		}
	}
}
