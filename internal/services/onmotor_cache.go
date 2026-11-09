package services

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/repository"
)

// tracked guarda uma linha do lote: como estava no banco (base) e se o lote mexeu nela.
type tracked[T any, S comparable] struct {
	row   *T
	base  S
	isNew bool
	dirty bool
}

type (
	trackedCustomer = tracked[models.Customer, customerSnapshot]
	trackedVehicle  = tracked[models.CustomerVehicle, vehicleSnapshot]
	trackedService  = tracked[models.ServiceRecord, serviceSnapshot]
)

// importCache é o retrato em memória do que o lote pode tocar: carregado em poucas consultas,
// alterado registro a registro e gravado em lote (flush) no fim de cada etapa.
type importCache struct {
	customers        []*trackedCustomer
	customerBySource map[string]*trackedCustomer
	customerByCPF    map[string]*trackedCustomer
	customerByCNPJ   map[string]*trackedCustomer
	customerByMobile map[string][]*trackedCustomer
	// Chaves de cliente presentes no lote: um cadastro com uma delas ainda existe no OnMotor
	// e não pode ser "adotado" por outra chave.
	batchCustomerKeys map[string]bool

	vehicles        []*trackedVehicle
	vehicleBySource map[string]*trackedVehicle
	vehicleByPlate  map[string]*trackedVehicle

	services        []*trackedService
	serviceBySource map[string]*trackedService
}

func addKey(set map[string]bool, key string) {
	if key != "" {
		set[key] = true
	}
}

func keys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}

func validOrEmpty(f FlexString, normalize func(string) (string, error)) string {
	if !f.Set {
		return ""
	}
	v, err := normalize(f.Value)
	if err != nil {
		return ""
	}
	return v
}

// loadImportCache pré-carrega clientes, veículos e OS do lote (três consultas).
func loadImportCache(store OnMotorStore, wsID uint, in ImportInput) (*importCache, error) {
	c := &importCache{
		customerBySource: map[string]*trackedCustomer{}, customerByCPF: map[string]*trackedCustomer{},
		customerByCNPJ: map[string]*trackedCustomer{}, customerByMobile: map[string][]*trackedCustomer{},
		batchCustomerKeys: map[string]bool{},
		vehicleBySource:   map[string]*trackedVehicle{}, vehicleByPlate: map[string]*trackedVehicle{},
		serviceBySource: map[string]*trackedService{},
	}
	customerKeys, cpfs, cnpjs, mobiles := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	vehicleKeys, plates, serviceKeys := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, rec := range in.Customers {
		addKey(c.batchCustomerKeys, rec.SourceID.trimmed())
		addKey(customerKeys, rec.SourceID.trimmed())
		addKey(cpfs, validOrEmpty(rec.CPF, normalizeCPF))
		addKey(cnpjs, validOrEmpty(rec.CNPJ, normalizeCNPJ))
		addKey(mobiles, validOrEmpty(rec.Mobile, normalizeMobile))
	}
	for _, rec := range in.Vehicles {
		addKey(vehicleKeys, rec.SourceID.trimmed())
		addKey(customerKeys, rec.CustomerSourceID.trimmed())
		addKey(plates, normalizeCustomerPlate(rec.Plate.Value))
	}
	for _, rec := range in.Services {
		addKey(serviceKeys, rec.SourceID.trimmed())
		addKey(customerKeys, rec.CustomerSourceID.trimmed())
		addKey(vehicleKeys, rec.VehicleSourceID.trimmed())
		addKey(plates, normalizeCustomerPlate(rec.Plate.Value))
	}

	customers, err := store.CustomersForImport(wsID, repository.CustomerImportLookup{
		SourceIDs: keys(customerKeys), CPFs: keys(cpfs), CNPJs: keys(cnpjs), Mobiles: keys(mobiles),
	})
	if err != nil {
		return nil, err
	}
	for i := range customers {
		tc := &trackedCustomer{row: &customers[i], base: snapshotCustomer(&customers[i])}
		c.customers = append(c.customers, tc)
		c.indexCustomer(tc)
	}

	vehicles, err := store.VehiclesForImport(wsID, keys(vehicleKeys), keys(plates))
	if err != nil {
		return nil, err
	}
	for i := range vehicles {
		tv := &trackedVehicle{row: &vehicles[i], base: snapshotVehicle(&vehicles[i])}
		c.vehicles = append(c.vehicles, tv)
		c.indexVehicle(tv)
	}

	services, err := store.ServiceRecordsForImport(wsID, keys(serviceKeys))
	if err != nil {
		return nil, err
	}
	for i := range services {
		ts := &trackedService{row: &services[i], base: snapshotService(&services[i])}
		c.services = append(c.services, ts)
		c.serviceBySource[services[i].SourceID] = ts
	}
	return c, nil
}

func (c *importCache) newCustomer(wsID uint) *trackedCustomer {
	tc := &trackedCustomer{row: &models.Customer{WorkspaceID: wsID}, isNew: true}
	c.customers = append(c.customers, tc)
	return tc
}

func (c *importCache) newVehicle(wsID uint) *trackedVehicle {
	tv := &trackedVehicle{row: &models.CustomerVehicle{WorkspaceID: wsID}, isNew: true}
	c.vehicles = append(c.vehicles, tv)
	return tv
}

func (c *importCache) newService(wsID uint, sourceID string) *trackedService {
	ts := &trackedService{row: &models.ServiceRecord{WorkspaceID: wsID, Source: models.SourceOnMotor, SourceID: sourceID, Items: "[]"}, isNew: true}
	c.services = append(c.services, ts)
	c.serviceBySource[sourceID] = ts
	return ts
}

// Índices: o primeiro dono de uma chave fica (ordem por id); unindex só remove o que aponta para tc.
func (c *importCache) indexCustomer(tc *trackedCustomer) {
	r := tc.row
	if r.SourceID != nil && r.Source == models.SourceOnMotor {
		setIfAbsent(c.customerBySource, *r.SourceID, tc)
	}
	if r.CPF != "" {
		setIfAbsent(c.customerByCPF, r.CPF, tc)
	}
	if r.CNPJ != "" {
		setIfAbsent(c.customerByCNPJ, r.CNPJ, tc)
	}
	if r.Mobile != "" {
		c.customerByMobile[r.Mobile] = append(c.customerByMobile[r.Mobile], tc)
	}
}

func (c *importCache) unindexCustomer(tc *trackedCustomer) {
	r := tc.row
	if r.SourceID != nil {
		deleteIfOwner(c.customerBySource, *r.SourceID, tc)
	}
	deleteIfOwner(c.customerByCPF, r.CPF, tc)
	deleteIfOwner(c.customerByCNPJ, r.CNPJ, tc)
	list := c.customerByMobile[r.Mobile]
	for i, other := range list {
		if other == tc {
			c.customerByMobile[r.Mobile] = append(list[:i:i], list[i+1:]...)
			break
		}
	}
}

func (c *importCache) indexVehicle(tv *trackedVehicle) {
	if tv.row.SourceID != nil && tv.row.Source == models.SourceOnMotor {
		setIfAbsent(c.vehicleBySource, *tv.row.SourceID, tv)
	}
	if tv.row.Plate != "" {
		setIfAbsent(c.vehicleByPlate, tv.row.Plate, tv)
	}
}

func (c *importCache) unindexVehicle(tv *trackedVehicle) {
	if tv.row.SourceID != nil {
		deleteIfOwner(c.vehicleBySource, *tv.row.SourceID, tv)
	}
	deleteIfOwner(c.vehicleByPlate, tv.row.Plate, tv)
}

func setIfAbsent[V comparable](m map[string]V, key string, v V) {
	var zero V
	if m[key] == zero {
		m[key] = v
	}
}

func deleteIfOwner[V comparable](m map[string]V, key string, v V) {
	if m[key] == v {
		delete(m, key)
	}
}

// normalizeName compara nomes sem acento, pontuação, caixa e espaços extras.
func normalizeName(name string) string {
	var b strings.Builder
	space := false
	for _, r := range norm.NFD.String(strings.ToLower(name)) {
		switch {
		case unicode.Is(unicode.Mn, r):
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if space && b.Len() > 0 {
				b.WriteByte(' ')
			}
			space = false
			b.WriteRune(r)
		default:
			space = true
		}
	}
	return b.String()
}

// flushCustomers grava as mudanças dos clientes: solta chaves que mudaram, insere os novos,
// regrava os alterados e só marca synced_at nos que vieram iguais.
func (imp *importer) flushCustomers() error {
	var creates, updates []*models.Customer
	var release, touch []uint
	for _, tc := range imp.cache.customers {
		if !tc.dirty {
			continue
		}
		r := tc.row
		r.SyncedAt = &imp.now
		switch now := snapshotCustomer(r); {
		case tc.isNew:
			creates = append(creates, r)
		case now != tc.base:
			updates = append(updates, r)
			if (tc.base.cpf != "" && now.cpf != tc.base.cpf) || (tc.base.sourceID != "" && now.sourceID != tc.base.sourceID) {
				release = append(release, r.ID)
			}
		default:
			touch = append(touch, r.ID)
		}
	}
	steps := []func() error{
		func() error { return imp.store.ReleaseCustomerKeys(release) },
		func() error { return imp.store.CreateCustomers(creates) },
		func() error { return imp.store.UpsertCustomers(updates) },
		func() error { return imp.store.TouchCustomers(touch, imp.now) },
	}
	if err := runSteps(steps); err != nil {
		return err
	}
	for _, tc := range imp.cache.customers {
		if tc.dirty {
			tc.isNew, tc.dirty, tc.base = false, false, snapshotCustomer(tc.row)
		}
	}
	return nil
}

func (imp *importer) flushVehicles() error {
	var creates, updates []*models.CustomerVehicle
	var release, touch []uint
	for _, tv := range imp.cache.vehicles {
		if !tv.dirty {
			continue
		}
		r := tv.row
		r.SyncedAt = &imp.now
		switch now := snapshotVehicle(r); {
		case tv.isNew:
			creates = append(creates, r)
		case now != tv.base:
			updates = append(updates, r)
			if (tv.base.plate != "" && now.plate != tv.base.plate) || (tv.base.sourceID != "" && now.sourceID != tv.base.sourceID) {
				release = append(release, r.ID)
			}
		default:
			touch = append(touch, r.ID)
		}
	}
	steps := []func() error{
		func() error { return imp.store.ReleaseVehicleKeys(release) },
		func() error { return imp.store.CreateVehicles(creates) },
		func() error { return imp.store.UpsertVehicles(updates) },
		func() error { return imp.store.TouchVehicles(touch, imp.now) },
	}
	if err := runSteps(steps); err != nil {
		return err
	}
	for _, tv := range imp.cache.vehicles {
		if tv.dirty {
			tv.isNew, tv.dirty, tv.base = false, false, snapshotVehicle(tv.row)
		}
	}
	return nil
}

func (imp *importer) flushServices() error {
	var creates, updates []*models.ServiceRecord
	var touch []uint
	for _, ts := range imp.cache.services {
		if !ts.dirty {
			continue
		}
		r := ts.row
		r.SyncedAt = &imp.now
		switch {
		case ts.isNew:
			creates = append(creates, r)
		case snapshotService(r) != ts.base:
			updates = append(updates, r)
		default:
			touch = append(touch, r.ID)
		}
	}
	steps := []func() error{
		func() error { return imp.store.CreateServiceRecords(creates) },
		func() error { return imp.store.UpsertServiceRecords(updates) },
		func() error { return imp.store.TouchServiceRecords(touch, imp.now) },
	}
	if err := runSteps(steps); err != nil {
		return err
	}
	for _, ts := range imp.cache.services {
		if ts.dirty {
			ts.isNew, ts.dirty, ts.base = false, false, snapshotService(ts.row)
		}
	}
	return nil
}

func runSteps(steps []func() error) error {
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}
