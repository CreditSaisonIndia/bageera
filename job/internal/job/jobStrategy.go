package job

/*
	Represents the stategy type like insertion/updation/deletion.

*/

type JobStrategy interface {

	/*
		Below method should be implemented by concerete structs with their own use case
	*/
	ExecuteJob(path string, tableName string)

	GetFileNamePattern() string

	SetFileNamePattern(pattern string)
}
