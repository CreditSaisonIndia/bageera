package job

import (
	"fmt"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/job/deletion"
	"github.com/CreditSaisonIndia/bageera/internal/job/insertion"
	"github.com/CreditSaisonIndia/bageera/internal/job/migration"
	"github.com/CreditSaisonIndia/bageera/internal/job/updation"
	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
)

type Job struct {
	jobStrategy JobStrategy
}

func SetStrategy(jobStrategy JobStrategy) *Job {
	return &Job{
		jobStrategy: jobStrategy,
	}
}

func (j *Job) ExecuteStrategy(path string, tableName string) {
	j.jobStrategy.ExecuteJob(path, tableName)
}

func GetJob() (*Job, error) {
	LOGGER := customLogger.GetLogger()
	LOGGER.Infof(" FOUND %s Type", serviceConfig.ApplicationSetting.JobType)
	switch serviceConfig.ApplicationSetting.JobType {
	case "insert":
		insertionJob := &insertion.Insertion{}
		insertionJob.SetFileNamePattern(`.*_\d+_exist_failure\.csv`)
		return SetStrategy(insertionJob), nil

	case "delete":
		deletionJob := &deletion.Deletion{}
		deletionJob.SetFileNamePattern(`.*_\d+_exist_success\.csv`)
		return SetStrategy(deletionJob), nil

	case "update":
		updationJob := &updation.Updation{}
		updationJob.SetFileNamePattern(`.*_\d+_exist_success\.csv`)
		return SetStrategy(updationJob), nil

	case "migrate":
		migrationJob := &migration.Migration{}
		migrationJob.SetFileNamePattern("")
		return SetStrategy(migrationJob), nil

	default:
		LOGGER.Error("INVALID JOB TYPE")
		return nil, fmt.Errorf("Unsupported Job Type Found : %s", serviceConfig.ApplicationSetting.JobType)
	}
}

func (j *Job) SetStrategyFilePattern(pattern string) {
	j.jobStrategy.SetFileNamePattern(pattern)
}

func (j *Job) GetStrategyFilePattern() string {
	return j.jobStrategy.GetFileNamePattern()
}
