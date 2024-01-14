package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds/rdsutils"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
)

type Peer struct {
	Name              string
	BbPool            *pgxpool.Pool
	Weight            int
	Logger            *zap.SugaredLogger
	Mu                sync.Mutex
	IAMRoleAuth       bool
	expire            time.Time
	cachedCredentials cachedCredentials
	tokenMu           sync.Mutex
}

type cachedCredentials struct {
	user     string
	password string
	host     string
	port     int
}

type DBConfig struct {
	Host        string
	Port        int
	User        string
	Password    string
	SSLMode     string
	Name        string
	MinConn     int
	MaxConn     int
	LifeTime    string
	IdleTime    string
	LogLevel    string
	Region      string
	IAMRoleAuth bool
}

var pgxPool *pgxpool.Pool

func GetPgxPool() *pgxpool.Pool {
	return pgxPool
}

func (p *Peer) isTokenExpired() bool {
	p.tokenMu.Lock()
	defer p.tokenMu.Unlock()

	if p.expire == (time.Time{}) {
		return true
	}
	return time.Since(p.expire) > 13*time.Minute

}

// getDBPool returns a new pgxpool.Pool instance.
// If peer IAMRoleAuth is true then BeforeConnect method is implemented
// BeforeConnect() is used to inject the authToken before a connection is made.
// Connection `LifeTime` is set to 14 mins, hence the connection will expire automatically and no intervention is needed to close the connection.
func (p *Peer) GetDBPool(ctx context.Context, cfg DBConfig, sess *session.Session) (*pgxpool.Pool, error) {

	poolCfg, err := pgxpool.ParseConfig(getDBURL(cfg))
	if err != nil {
		p.Logger.Error(err)
		p.Logger.Info("unable to parse config for peer: %v cfg: %v", cfg.Name, cfg)
		return nil, err
	}

	poolCfg.MaxConnIdleTime = 5 * time.Minute
	poolCfg.MaxConnLifetime = 13 * time.Minute
	poolCfg.MaxConns = 20
	poolCfg.MinConns = 5

	poolCfg.BeforeConnect = func(ctx context.Context, config *pgx.ConnConfig) error {
		// p.Mu.Unlock()
		// defer p.Mu.Unlock()
		p.Logger.Info("RDS Credential beforeConnect(), creating new credential")
		if p.isTokenExpired() {
			p.Logger.Info("********** EXPIRED HENCE GETTING TOKEN AGAIN *********")
			token, err := p.getCredential(poolCfg, cfg, sess)
			if err != nil {
				return err
			}

			config.User = cfg.User
			config.Password = token
			config.Host = cfg.Host
			config.Database = cfg.Name
			config.Port = 5432
			cachedCredentials := cachedCredentials{
				user:     cfg.User,
				password: token,
				host:     cfg.Host,
				port:     5432,
			}
			p.cachedCredentials = cachedCredentials
			p.expire = time.Now()
		} else {
			p.Logger.Info("********** NOT EXPIRED HENCE NOT GETTING TOKEN ************")
			config.User = p.cachedCredentials.user
			config.Password = p.cachedCredentials.password
			config.Host = p.cachedCredentials.host
			config.Database = cfg.Name
			config.Port = 5432
		}

		// if time.Since(p.lastTokenTime) > 13*time.Minute {
		// 	p.Logger.Info("TIME LIMIT REACHED ")
		// 	newPassword, err := p.getCredential(poolCfg, cfg, sess)
		// 	if err != nil {
		// 		return err
		// 	}
		// 	p.Mu.Lock()
		// 	config.Password = newPassword
		// 	p.lastTokenTime = time.Now()
		// 	p.Mu.Unlock()
		// 	p.Logger.Info("BeforeConnect | CONNECTION STRING: ", poolCfg.ConnConfig.ConnString())
		// }

		return nil
	}

	p.Logger.Info("GetDBPool | CONNECTION STRING: ", poolCfg.ConnConfig.ConnString())
	pool, err := pgxpool.ConnectConfig(ctx, poolCfg)
	if err != nil {
		p.Logger.Error("unable to connect to db : ", err)
		return nil, err
	}
	p.Logger.Info("PGX pool setup successfully done")
	pgxPool = pool
	return pool, nil
}

// getCredential returns the new password to connect to RDS
func (p *Peer) getCredential(poolCfg *pgxpool.Config, cfg DBConfig, sess *session.Session) (string, error) {
	dbEndpoint := fmt.Sprintf("%s:%d", poolCfg.ConnConfig.Host, poolCfg.ConnConfig.Port)
	awsRegion := cfg.Region
	dbUser := poolCfg.ConnConfig.User
	authToken, err := rdsutils.BuildAuthToken(dbEndpoint, awsRegion, dbUser, sess.Config.Credentials)
	if err != nil {
		p.Logger.Error("Error in building auth token to connect with RDS: ", err)
		return "", err
	}
	p.Logger.Info("authToken : ", authToken)
	return authToken, nil
}

func getDBURL(cfg DBConfig) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s&search_path=scarlet",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode)
}
