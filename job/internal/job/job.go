package job

type Job struct {
	jobStrategy JobStrategy
}

func SetStrategy(jobStrategy JobStrategy) *Job {
	return &Job{
		jobStrategy: jobStrategy,
	}
}

func (j *Job) ExecuteStrategy(path string) {
	j.jobStrategy.Execute(path)
}
