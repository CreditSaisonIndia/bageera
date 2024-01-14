package dbmanager

import (
	"fmt"
	"sync"
	"time"

	"github.com/CreditSaisonIndia/bageera/internal/database"
	"gorm.io/gorm"
)

type CustomDB struct {
	*gorm.DB
	LastDBUsage time.Time
}

// CustomDBManager manages the CustomDB instances
type CustomDBManager struct {
	mu       sync.Mutex
	dbChan   chan *CustomDB
	timeout  time.Duration
	stopChan chan struct{}
}

// NewCustomDBManager creates a new CustomDBManager instance
func NewCustomDBManager(channelSize int, timeout time.Duration) *CustomDBManager {
	fmt.Println("NewCustomDBManager")
	return &CustomDBManager{
		dbChan:   make(chan *CustomDB, channelSize),
		timeout:  timeout,
		stopChan: make(chan struct{}),
	}
}

// Init initializes the CustomDBManager and starts the goroutine to maintain the channel
func (m *CustomDBManager) Init() {
	fmt.Println("Init")
	// Trigger checkAvailability once at the beginning
	m.checkAvailability()

	// // Start the cleanup goroutine
	// go m.cleanup()

	// // Start the periodic checkAvailability goroutine
	// go func() {
	// 	for {
	// 		select {
	// 		case <-time.After(1 * time.Minute):
	// 			m.checkAvailability()
	// 		case <-m.stopChan:
	// 			return // Stop the goroutine
	// 		}
	// 	}
	// }()
}

// Stop stops the cleanup goroutine and closes the database channel
func (m *CustomDBManager) Stop() {
	fmt.Println("Stop")
	close(m.stopChan)
	close(m.dbChan)
}

func (m *CustomDBManager) cleanup() {
	fmt.Println("cleanup")
	for {
		select {
		case <-time.After(m.timeout):
			m.mu.Lock()
			for len(m.dbChan) > 0 {
				dbInstance := <-m.dbChan
				fmt.Printf("Checking stale DB instance %p\n", dbInstance)

				// Simulate some work with the dbInstance
				// Check the last usage time
				lastUsage := dbInstance.LastDBUsage

				// Calculate the time since last usage
				elapsed := time.Since(lastUsage)

				if elapsed > m.timeout {
					fmt.Println("Removing stale DB instance")
					// Close the underlying database connection
					// if err := dbInstance.DB.c Close(); err != nil {
					// 	fmt.Printf("Error closing database connection: %v\n", err)
					// }
				} else {
					// Put it back in the channel if it's still valid
					m.dbChan <- dbInstance
					break
				}
			}
			m.mu.Unlock()
		case <-m.stopChan:
			return // Stop the goroutine
		}
	}
}

// func (m *CustomDBManager) checkAvailability() {
// 	fmt.Println("checkAvailability")
// 	for {
// 		select {
// 		case <-time.After(1 * time.Minute): // Adjust the interval as needed
// 			m.mu.Lock()
// 			currentInstances := len(m.dbChan)
// 			m.mu.Unlock()

// 			if currentInstances < 10/2 {
// 				// If the channel is running low, create and add new instances
// 				newInstances := 10 - currentInstances
// 				for i := 0; i < newInstances; i++ {
// 					db, err := database.InitGormDb()
// 					if err != nil {
// 						fmt.Printf("Error initializing database: %v\n", err)
// 						continue
// 					}

// 					cb := &CustomDB{
// 						DB:          db,
// 						LastDBUsage: time.Now(),
// 					}

// 					m.mu.Lock()
// 					m.dbChan <- cb
// 					m.mu.Unlock()
// 				}
// 			}
// 		case <-m.stopChan:
// 			return
// 		}
// 	}
// }

// func (m *CustomDBManager) checkAvailability() {
// 	fmt.Println("checkAvailability")

// 	m.mu.Lock()
// 	currentInstances := len(m.dbChan)
// 	m.mu.Unlock()

// 	if currentInstances < 10/2 {
// 		// If the channel is running low, create and add new instances
// 		newInstances := 10 - currentInstances
// 		for i := 0; i < newInstances; i++ {
// 			db, err := database.InitGormDb()
// 			if err != nil {
// 				fmt.Printf("Error initializing database: %v\n", err)
// 				continue
// 			}

// 			cb := &CustomDB{
// 				DB:          db,
// 				LastDBUsage: time.Now(),
// 			}

// 			m.mu.Lock()
// 			m.dbChan <- cb
// 			m.mu.Unlock()
// 		}
// 	}
// }

func (m *CustomDBManager) checkAvailability() {
	fmt.Println("checkAvailability")

	m.mu.Lock()
	defer m.mu.Unlock()
	db, err := database.InitGormDb()
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)

	}

	cb := &CustomDB{
		DB:          db,
		LastDBUsage: time.Now(),
	}

	m.mu.Lock()
	m.dbChan <- cb
	m.mu.Unlock()
}

// GetDB retrieves a CustomDB instance from the channel
func (m *CustomDBManager) GetDB() *CustomDB {
	fmt.Println("GetDB")
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.dbChan) > 0 {
		dbInstance := <-m.dbChan
		fmt.Printf("Got DB instance %p\n", dbInstance)
		return dbInstance
	} else {
		fmt.Println("No available DB instance in the channel.")
		return nil
	}
}

func (m *CustomDBManager) ReleaseDB(dbInstance *CustomDB) {
	fmt.Println("ReleaseDB")
	m.mu.Lock()
	defer m.mu.Unlock()

	if dbInstance == nil {
		fmt.Println("Trying to release a nil DB instance.")
		return
	}
	// Update the last usage time
	dbInstance.LastDBUsage = time.Now()

	m.dbChan <- dbInstance
	fmt.Printf("Released DB instance %p\n", dbInstance)
}

var GlobalDBManager *CustomDBManager

func InitDBManager() {
	GlobalDBManager = NewCustomDBManager(10, 10*time.Minute)
	GlobalDBManager.Init()
}
