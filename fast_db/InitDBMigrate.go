package fast_db

import (
	"database/sql"
	"fast_base"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func migrateDB() {

	m, err := migrate.New(fmt.Sprintf("file://%s", "./conf/db/migration"), "mysql://"+ConfigDataSource.DNS())
	defer func() {
		m.Close()
	}()

	if err != nil {
		fast_base.Logger.Fatal("migration failed...\n" + err.Error())
		panic("migration failed..." + err.Error())
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		fast_base.Logger.Fatal("An error occurred while syncing the database...\n" + err.Error())
		panic("An error occurred while syncing the database...\n" + err.Error())
	}
}

func migrateDBStepByStep() {
	// 连接数据库
	db, err := sql.Open("mysql", ConfigDataSource.DNS())
	defer func() {
		db.Close()
	}()

	if err != nil {
		fast_base.Logger.Fatal("could not ping DB...\n" + err.Error())
		panic("could not ping DB..." + err.Error())
	}
	if err := db.Ping(); err != nil {
		fast_base.Logger.Fatal("could not ping DB...\n" + err.Error())
		panic("could not ping DB..." + err.Error())
	}

	// 开始迁移
	driver, _ := mysql.WithInstance(db, &mysql.Config{})
	defer func() {
		driver.Close()
	}()
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", "./conf/db/migration"), // file://path/to/directory
		"mysql", driver)
	if err != nil {
		fast_base.Logger.Fatal("migration failed...\n" + err.Error())
		panic("migration failed..." + err.Error())
	}
	defer func() {
		m.Close()
	}()

	// 执行操作: up --> 更新,  down --> 回滚
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		fast_base.Logger.Fatal("An error occurred while syncing the database...\n" + err.Error())
		panic("An error occurred while syncing the database...\n" + err.Error())
	}
}
