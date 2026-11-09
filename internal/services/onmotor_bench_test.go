package services

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/repository"
)

// Benchmark manual da importação (não roda no verify):
//
//	ONMOTOR_BENCH=1 go test ./internal/services -run TestOnMotorImportBenchmark -v
//	BENCH_LATENCY_MS=20: soma 20ms a cada statement do GORM (não vê BEGIN/COMMIT/prepare).
//	BENCH_PG_DSN="postgres://..." [BENCH_RTT_MS=20]: Postgres; com BENCH_RTT_MS, passa por um proxy TCP
//	que atrasa cada pacote em RTT/2 em cada sentido (mede as idas reais ao banco, como no Neon).
func TestOnMotorImportBenchmark(t *testing.T) {
	if os.Getenv("ONMOTOR_BENCH") == "" {
		t.Skip("defina ONMOTOR_BENCH=1 para rodar o benchmark da importação")
	}
	latency, _ := strconv.Atoi(os.Getenv("BENCH_LATENCY_MS"))
	for _, n := range []int{100, 500} {
		db := benchDB(t)
		var statements int64
		countStatements(db, &statements, time.Duration(latency)*time.Millisecond)
		var customers int64
		svc := NewOnMotorSyncService(repository.NewCustomerRepository(db), repository.NewIntegrationRepository(db), fixtureWorkspaceStore())
		token := &models.IntegrationToken{ID: 1, WorkspaceID: fxWorkspaceID}

		for _, scenario := range []struct {
			name string
			in   ImportInput
		}{
			{"novo", benchBatch(n, 0, "São Paulo", "Prata")},
			{"reenvio igual", benchBatch(n, 0, "São Paulo", "Prata")},
			{"tudo alterado", benchBatch(n, 0, "Campinas", "Preto")},
			{"novo, chave nome", benchNameKeys(n, n)},
			{"nome → cpf", benchBatch(n, n, "Campinas", "Preto")},
		} {
			atomic.StoreInt64(&statements, 0)
			start := time.Now()
			res, err := svc.ImportWithToken(token, scenario.in)
			require.NoError(t, err)
			elapsed := time.Since(start)
			q := atomic.LoadInt64(&statements)
			db.Session(&gorm.Session{SkipHooks: true}).Model(&models.Customer{}).Count(&customers)
			t.Logf("%d clientes + %d veículos | %-16s | %5d statements | %8s | estimativa a 20ms: %6.1fs | %+v",
				n, n, scenario.name, q, elapsed.Round(time.Millisecond), float64(q)*0.02, res.Counters)
			t.Logf("    clientes na base: %d | avisos: %v", customers, res.Warnings)
		}
	}
}

func benchDB(t *testing.T) *gorm.DB {
	t.Helper()
	all := []any{
		&models.User{}, &models.Workspace{}, &models.WorkspaceMember{}, &models.WorkspaceInvite{},
		&models.TaskList{}, &models.TaskItem{}, &models.ListAssignment{}, &models.QuoteItem{},
		&models.PendingFlag{}, &models.VehicleNote{}, &models.TokenBlacklist{}, &models.AuditLog{},
		&models.Customer{}, &models.CustomerVehicle{}, &models.ServiceRecord{}, &models.SyncRun{}, &models.IntegrationToken{},
	}
	cfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	if dsn := os.Getenv("BENCH_PG_DSN"); dsn != "" {
		if rtt, _ := strconv.Atoi(os.Getenv("BENCH_RTT_MS")); rtt > 0 {
			dsn = delayedDSN(t, dsn, time.Duration(rtt)*time.Millisecond/2)
		}
		db, err := gorm.Open(postgres.Open(dsn), cfg)
		require.NoError(t, err)
		require.NoError(t, db.Migrator().DropTable(all...))
		require.NoError(t, db.AutoMigrate(all...))
		return db
	}
	db, err := gorm.Open(sqlite.Open(":memory:"), cfg)
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(all...))
	return db
}

// countStatements conta cada ida ao banco feita pelo GORM e, se pedido, simula a latência de rede.
func countStatements(db *gorm.DB, counter *int64, latency time.Duration) {
	hook := func(*gorm.DB) {
		atomic.AddInt64(counter, 1)
		if latency > 0 {
			time.Sleep(latency)
		}
	}
	cb := db.Callback()
	_ = cb.Create().Before("gorm:create").Register("bench:create", hook)
	_ = cb.Query().Before("gorm:query").Register("bench:query", hook)
	_ = cb.Update().Before("gorm:update").Register("bench:update", hook)
	_ = cb.Delete().Before("gorm:delete").Register("bench:delete", hook)
	_ = cb.Row().Before("gorm:row").Register("bench:row", hook)
	_ = cb.Raw().Before("gorm:raw").Register("bench:raw", hook)
}

// delayedDSN sobe um proxy TCP local que atrasa cada pacote e devolve o DSN apontando para ele.
func delayedDSN(t *testing.T, dsn string, delay time.Duration) string {
	t.Helper()
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	target := u.Host
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	pipe := func(dst, src net.Conn) {
		buf := make([]byte, 64*1024)
		for {
			n, err := src.Read(buf)
			if n > 0 {
				time.Sleep(delay)
				if _, werr := dst.Write(buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				_ = dst.Close()
				return
			}
		}
	}
	go func() {
		for {
			client, err := ln.Accept()
			if err != nil {
				return
			}
			server, err := net.Dial("tcp", target)
			if err != nil {
				_ = client.Close()
				continue
			}
			go pipe(server, client)
			go pipe(client, server)
		}
	}()
	u.Host = ln.Addr().String()
	return u.String()
}

func benchCPF(i int) string {
	base := fmt.Sprintf("%09d", 100000000+i)
	d1 := cpfCheckDigit(base)
	return base + string(d1) + string(cpfCheckDigit(base+string(d1)))
}

// benchNameKeys cria clientes sem CPF (chave nome+celular); o cenário seguinte manda os mesmos
// com CPF (chave cpf:...) e deve atualizar os mesmos ids, sem duplicar.
func benchNameKeys(n, offset int) ImportInput {
	in := ImportInput{Final: true}
	for i := offset; i < offset+n; i++ {
		mobile := fmt.Sprintf("119%08d", i)
		key := fmt.Sprintf("nome:cliente %d|cel:%s", i, mobile)
		in.Customers = append(in.Customers, ImportCustomer{SourceID: Flex(key), Name: Flex(fmt.Sprintf("Cliente %d", i)), Mobile: Flex(mobile)})
		in.Vehicles = append(in.Vehicles, ImportVehicle{SourceID: Flex(fmt.Sprintf("placa:BEN%04d", i)), CustomerSourceID: Flex(key), Plate: Flex(fmt.Sprintf("BEN%04d", i))})
	}
	return in
}

func benchBatch(n, offset int, city, color string) ImportInput {
	in := ImportInput{Final: true}
	for i := offset; i < offset+n; i++ {
		cpf := benchCPF(i)
		key := "cpf:" + cpf
		in.Customers = append(in.Customers, ImportCustomer{
			SourceID: Flex(key), Name: Flex(fmt.Sprintf("Cliente %d", i)), CPF: Flex(cpf),
			Mobile: Flex(fmt.Sprintf("119%08d", i)), Email: Flex(fmt.Sprintf("c%d@exemplo.com", i)),
			City: Flex(city), UF: Flex("SP"), LastVisitAt: Flex("2026-11-07T14:32:00-03:00"),
		})
		in.Vehicles = append(in.Vehicles, ImportVehicle{
			SourceID: Flex(fmt.Sprintf("placa:BEN%04d", i)), CustomerSourceID: Flex(key),
			Plate: Flex(fmt.Sprintf("BEN%04d", i)), Model: Flex("Gol"), YearModel: Flex("2015"), Color: Flex(color),
		})
	}
	return in
}
