package services

import (
	"fmt"
	"time"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

// customerSnapshot: campos que contam como "mudou" (para updated_at e contadores).
type customerSnapshot struct {
	source, sourceID, name, personType, cpf, cnpj, rg, phone, mobile, email, emailSecondary string
	cep, street, number, complement, district, city, uf, birthDate, notes                   string
	lastVisit, sourceUpdated                                                                int64
}

func snapshotCustomer(c *models.Customer) customerSnapshot {
	s := customerSnapshot{
		source: c.Source, name: c.Name, personType: c.PersonType, cpf: c.CPF, cnpj: c.CNPJ, rg: c.RG,
		phone: c.Phone, mobile: c.Mobile, email: c.Email, emailSecondary: c.EmailSecondary, cep: c.CEP,
		street: c.Street, number: c.Number, complement: c.Complement, district: c.District, city: c.City,
		uf: c.UF, notes: c.Notes, lastVisit: optTime(c.LastVisitAt), sourceUpdated: optTime(c.SourceUpdatedAt),
	}
	if c.SourceID != nil {
		s.sourceID = *c.SourceID
	}
	if c.BirthDate != nil {
		s.birthDate = c.BirthDate.String()
	}
	return s
}

// isOnMotorID: só ids numéricos do OnMotor aparecem nos avisos. Chaves derivadas
// (cpf:..., nome:...|cel:..., placa:...) contêm dados pessoais e nunca são citadas.
func isOnMotorID(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func recordLabel(kind string, pos int, sourceID string) string {
	if isOnMotorID(sourceID) {
		return kind + " OnMotor " + sourceID
	}
	return fmt.Sprintf("%s na posição %d do lote", kind, pos)
}

func referenceLabel(kind, sourceID string) string {
	if isOnMotorID(sourceID) {
		return kind + " OnMotor " + sourceID
	}
	return kind + " referenciado"
}

func describeCustomer(c *models.Customer) string {
	switch {
	case c.SourceID != nil && isOnMotorID(*c.SourceID):
		return "OnMotor " + *c.SourceID
	case c.ID == 0:
		return "outro cliente do lote"
	case c.Source == models.SourceLocal:
		return fmt.Sprintf("local #%d", c.ID)
	}
	return fmt.Sprintf("#%d", c.ID)
}

func describeVehicle(v *models.CustomerVehicle) string {
	switch {
	case v.SourceID != nil && isOnMotorID(*v.SourceID):
		return "OnMotor " + *v.SourceID
	case v.ID == 0:
		return "outro veículo do lote"
	case v.Source == models.SourceLocal:
		return fmt.Sprintf("local #%d", v.ID)
	}
	return fmt.Sprintf("#%d", v.ID)
}

// matchCustomer procura, para uma chave ainda desconhecida, um cadastro da mesma pessoa:
// (a) mesmo CPF ou CNPJ; (b) cliente do OnMotor sem CPF/CNPJ com mesmo nome normalizado e celular.
// Cadastro cuja chave atual também está no lote é outro registro vivo do OnMotor: não é adotado.
func (imp *importer) matchCustomer(rec ImportCustomer) *trackedCustomer {
	c := imp.cache
	usable := func(tc *trackedCustomer) bool {
		return tc != nil && (tc.row.SourceID == nil || !c.batchCustomerKeys[*tc.row.SourceID])
	}
	if cpf := validOrEmpty(rec.CPF, normalizeCPF); cpf != "" && usable(c.customerByCPF[cpf]) {
		return c.customerByCPF[cpf]
	}
	if cnpj := validOrEmpty(rec.CNPJ, normalizeCNPJ); cnpj != "" && usable(c.customerByCNPJ[cnpj]) {
		return c.customerByCNPJ[cnpj]
	}
	mobile, name := validOrEmpty(rec.Mobile, normalizeMobile), normalizeName(rec.Name.Value)
	if mobile == "" || name == "" {
		return nil
	}
	for _, tc := range c.customerByMobile[mobile] {
		r := tc.row
		if usable(tc) && r.Source == models.SourceOnMotor && r.CPF == "" && r.CNPJ == "" && normalizeName(r.Name) == name {
			return tc
		}
	}
	return nil
}

func (imp *importer) customer(pos int, rec ImportCustomer) {
	sourceID := rec.SourceID.trimmed()
	if sourceID == "" {
		imp.skip("cliente na posição %d sem source_id (ignorado)", pos)
		return
	}
	label := recordLabel("cliente", pos, sourceID)

	tc := imp.cache.customerBySource[sourceID]
	if tc == nil {
		if tc = imp.matchCustomer(rec); tc != nil && tc.row.SourceID != nil {
			imp.keyChanges = append(imp.keyChanges, tc)
		}
	}
	isNew := tc == nil
	if isNew {
		if rec.Name.trimmed() == "" {
			imp.skip("%s sem nome (ignorado)", label)
			return
		}
		tc = imp.cache.newCustomer(imp.wsID)
	}
	imp.cache.unindexCustomer(tc)
	defer imp.cache.indexCustomer(tc)
	tc.dirty = true
	target := tc.row
	before := snapshotCustomer(target)
	// A chave nova substitui a antiga no mesmo cadastro (id mantido: veículos e atendimentos seguem ligados).
	target.Source = models.SourceOnMotor
	target.SourceID = &sourceID

	if !isNew && notNewer(rec.SourceUpdatedAt, target.SourceUpdatedAt) {
		imp.customerVisitOnly(target, before, rec.LastVisitAt)
		return
	}

	if name := rec.Name.trimmed(); name != "" {
		target.Name = name
	}
	imp.normalized(label, "tipo de pessoa", rec.PersonType, normalizePersonType, &target.PersonType)
	imp.normalized(label, "CPF", rec.CPF, normalizeCPF, &target.CPF)
	imp.normalized(label, "CNPJ", rec.CNPJ, normalizeCNPJ, &target.CNPJ)
	imp.normalized(label, "telefone", rec.Phone, normalizePhone, &target.Phone)
	imp.normalized(label, "celular", rec.Mobile, normalizeMobile, &target.Mobile)
	imp.normalized(label, "e-mail", rec.Email, normalizeEmail, &target.Email)
	imp.normalized(label, "e-mail secundário", rec.EmailSecondary, normalizeEmail, &target.EmailSecondary)
	imp.normalized(label, "CEP", rec.CEP, normalizeCEP, &target.CEP)
	imp.normalized(label, "UF", rec.UF, normalizeUF, &target.UF)
	for _, f := range []struct {
		in     FlexString
		target *string
	}{
		{rec.RG, &target.RG}, {rec.Street, &target.Street}, {rec.Number, &target.Number},
		{rec.Complement, &target.Complement}, {rec.District, &target.District},
		{rec.City, &target.City}, {rec.Notes, &target.Notes},
	} {
		plainField(f.in, f.target)
	}
	imp.timeField(label, "last_visit_at", rec.LastVisitAt, &target.LastVisitAt)
	imp.timeField(label, "source_updated_at", rec.SourceUpdatedAt, &target.SourceUpdatedAt)
	if rec.BirthDate.Set {
		d, err := parseBirthDate(rec.BirthDate.Value, imp.now)
		if err != nil {
			imp.warn("%s: data de nascimento inválida (campo ignorado)", label)
		}
		target.BirthDate = d
	}

	// CPF já usado por outro cliente da oficina: fica com o primeiro; este fica sem CPF.
	if target.CPF != "" {
		if holder := imp.cache.customerByCPF[target.CPF]; holder != nil && holder != tc {
			imp.warn("CPF duplicado em 2 clientes (%s, %s); mantido no primeiro", describeCustomer(holder.row), label)
			target.CPF = ""
		}
	}

	switch {
	case isNew:
		imp.run.CustomersCreated++
	case snapshotCustomer(target) != before:
		imp.run.CustomersUpdated++
	}
}

// notNewer: a "Data de edição" recebida não é posterior à já gravada (cadastro sem edição nova no OnMotor).
func notNewer(incoming FlexString, stored *time.Time) bool {
	if !incoming.Set || stored == nil {
		return false
	}
	t, err := parseImportTime(incoming.Value)
	return err == nil && t != nil && !t.After(*stored)
}

// customerVisitOnly: sem edição nova, os dados cadastrais ficam como estão; só a última visita
// avança (ela muda no OnMotor sem alterar a data de edição do cadastro).
func (imp *importer) customerVisitOnly(c *models.Customer, before customerSnapshot, lastVisit FlexString) {
	if lastVisit.Set {
		if t, err := parseImportTime(lastVisit.Value); err == nil && t != nil && (c.LastVisitAt == nil || t.After(*c.LastVisitAt)) {
			c.LastVisitAt = t
		}
	}
	if snapshotCustomer(c) != before {
		imp.run.CustomersUpdated++
	}
}

// warnKeyChanges resume as trocas de chave do lote (ids locais não são dado pessoal).
func (imp *importer) warnKeyChanges() {
	if len(imp.keyChanges) == 0 {
		return
	}
	ids := ""
	for i, tc := range imp.keyChanges {
		if i == 20 {
			ids += ", …"
			break
		}
		if i > 0 {
			ids += ", "
		}
		ids += fmt.Sprintf("#%d", tc.row.ID)
	}
	imp.warn("chave de cliente atualizada (%d): %s", len(imp.keyChanges), ids)
	imp.keyChanges = nil
}

type vehicleSnapshot struct {
	source, sourceID, plate, manufacturer, model, transmission, fuel, engine, color, chassis, renavam string
	customerID                                                                                        uint
	yearFab, yearModel, doors, lastMileage                                                            int64
}

func optInt(n *int) int64 {
	if n == nil {
		return -1
	}
	return int64(*n)
}

func snapshotVehicle(v *models.CustomerVehicle) vehicleSnapshot {
	s := vehicleSnapshot{
		source: v.Source, plate: v.Plate, manufacturer: v.Manufacturer, model: v.Model,
		transmission: v.Transmission, fuel: v.Fuel, engine: v.Engine, color: v.Color,
		chassis: v.Chassis, renavam: v.Renavam,
		yearFab: optInt(v.YearFab), yearModel: optInt(v.YearModel), doors: optInt(v.Doors), lastMileage: optInt(v.LastMileage),
	}
	if v.SourceID != nil {
		s.sourceID = *v.SourceID
	}
	if v.CustomerID != nil {
		s.customerID = *v.CustomerID
	}
	return s
}

// owner devolve o cliente do OnMotor pela chave (já gravado: o flush dos clientes vem antes).
func (imp *importer) owner(label, key string) *uint {
	tc := imp.cache.customerBySource[key]
	if tc == nil {
		imp.warn("%s: %s não encontrado", label, referenceLabel("cliente", key))
		return nil
	}
	id := tc.row.ID
	return &id
}

func (imp *importer) vehicle(pos int, rec ImportVehicle) {
	sourceID := rec.SourceID.trimmed()
	if sourceID == "" {
		imp.skip("veículo na posição %d sem source_id (ignorado)", pos)
		return
	}
	label := recordLabel("veículo", pos, sourceID)
	plate := normalizeCustomerPlate(rec.Plate.Value)

	tv := imp.cache.vehicleBySource[sourceID]
	if tv == nil && plate != "" {
		// Veículo local (sem source_id) com a mesma placa: vira o registro do OnMotor.
		if local := imp.cache.vehicleByPlate[plate]; local != nil && local.row.SourceID == nil {
			tv = local
		}
	}
	isNew := tv == nil
	if isNew {
		tv = imp.cache.newVehicle(imp.wsID)
	}
	imp.cache.unindexVehicle(tv)
	defer imp.cache.indexVehicle(tv)
	tv.dirty = true
	target := tv.row
	before := snapshotVehicle(target)
	target.Source = models.SourceOnMotor
	target.SourceID = &sourceID

	if key := rec.CustomerSourceID.trimmed(); key != "" {
		if id := imp.owner(label, key); id != nil {
			target.CustomerID = id
		}
	}
	for _, f := range []struct {
		in     FlexString
		target *string
	}{
		{rec.Manufacturer, &target.Manufacturer}, {rec.Model, &target.Model}, {rec.Transmission, &target.Transmission},
		{rec.Fuel, &target.Fuel}, {rec.Engine, &target.Engine}, {rec.Color, &target.Color},
		{rec.Chassis, &target.Chassis}, {rec.Renavam, &target.Renavam},
	} {
		plainField(f.in, f.target)
	}
	yearOK := func(n *int) bool { return validYear(n, imp.now) }
	imp.intField(label, "ano de fabricação", rec.YearFab, yearOK, &target.YearFab)
	imp.intField(label, "ano do modelo", rec.YearModel, yearOK, &target.YearModel)
	imp.intField(label, "portas", rec.Doors, validDoors, &target.Doors)
	var km *int
	imp.intField(label, "quilometragem", rec.LastMileage, validMileage, &km)
	if km != nil && (target.LastMileage == nil || *km > *target.LastMileage) {
		target.LastMileage = km
	}

	if rec.Plate.Set {
		target.Plate = plate
	}
	// Placa de outro veículo: a última gravação vence; o anterior fica sem placa.
	if target.Plate != "" {
		if holder := imp.cache.vehicleByPlate[target.Plate]; holder != nil && holder != tv {
			imp.cache.unindexVehicle(holder)
			holder.row.Plate = ""
			holder.dirty = true
			imp.cache.indexVehicle(holder)
			imp.warn("placa duplicada: %s assumiu a placa do veículo %s", label, describeVehicle(holder.row))
		}
	}

	switch {
	case isNew:
		imp.run.VehiclesCreated++
	case snapshotVehicle(target) != before:
		imp.run.VehiclesUpdated++
	}
}

type serviceSnapshot struct {
	customerID, vehicleID                 uint
	plate, kind, status, notes, items     string
	mileage                               int64
	total                                 float64
	hasTotal                              bool
	opened, diagnosed, quoted, started    int64
	tested, finished, notified, delivered int64
}

func optTime(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return t.UnixMicro()
}

func snapshotService(sr *models.ServiceRecord) serviceSnapshot {
	s := serviceSnapshot{
		plate: sr.Plate, kind: sr.Type, status: sr.Status, notes: sr.Notes, items: string(sr.Items),
		mileage: optInt(sr.Mileage),
		opened:  optTime(sr.OpenedAt), diagnosed: optTime(sr.DiagnosedAt), quoted: optTime(sr.QuotedAt),
		started: optTime(sr.StartedAt), tested: optTime(sr.TestedAt), finished: optTime(sr.FinishedAt),
		notified: optTime(sr.NotifiedAt), delivered: optTime(sr.DeliveredAt),
	}
	if sr.CustomerID != nil {
		s.customerID = *sr.CustomerID
	}
	if sr.CustomerVehicleID != nil {
		s.vehicleID = *sr.CustomerVehicleID
	}
	if sr.Total != nil {
		s.total, s.hasTotal = *sr.Total, true
	}
	return s
}

func (imp *importer) service(pos int, rec ImportService) {
	sourceID := rec.SourceID.trimmed()
	if sourceID == "" {
		imp.skip("OS na posição %d sem source_id (ignorada)", pos)
		return
	}
	label := recordLabel("OS", pos, sourceID)

	ts := imp.cache.serviceBySource[sourceID]
	isNew := ts == nil
	if isNew {
		ts = imp.cache.newService(imp.wsID, sourceID)
	}
	ts.dirty = true
	target := ts.row
	before := snapshotService(target)

	if key := rec.CustomerSourceID.trimmed(); key != "" {
		if id := imp.owner(label, key); id != nil {
			target.CustomerID = id
		}
	}

	plate := normalizeCustomerPlate(rec.Plate.Value)
	vehicle := imp.serviceVehicle(label, rec.VehicleSourceID.trimmed(), plate)
	if vehicle != nil {
		id := vehicle.row.ID
		target.CustomerVehicleID = &id
		if target.CustomerID == nil && vehicle.row.CustomerID != nil {
			owner := *vehicle.row.CustomerID
			target.CustomerID = &owner
		}
	}
	switch {
	case rec.Plate.Set:
		target.Plate = plate
	case target.Plate == "" && vehicle != nil:
		target.Plate = vehicle.row.Plate
	}

	imp.intField(label, "quilometragem", rec.Mileage, validMileage, &target.Mileage)
	plainField(rec.Type, &target.Type)
	plainField(rec.Status, &target.Status)
	plainField(rec.Notes, &target.Notes)
	for _, f := range []struct {
		name   string
		in     FlexString
		target **time.Time
	}{
		{"opened_at", rec.OpenedAt, &target.OpenedAt}, {"diagnosed_at", rec.DiagnosedAt, &target.DiagnosedAt},
		{"quoted_at", rec.QuotedAt, &target.QuotedAt}, {"started_at", rec.StartedAt, &target.StartedAt},
		{"tested_at", rec.TestedAt, &target.TestedAt}, {"finished_at", rec.FinishedAt, &target.FinishedAt},
		{"notified_at", rec.NotifiedAt, &target.NotifiedAt}, {"delivered_at", rec.DeliveredAt, &target.DeliveredAt},
	} {
		imp.timeField(label, f.name, f.in, f.target)
	}
	if itemsSent(rec.Items) {
		items, ok := normalizeItems(rec.Items)
		if !ok {
			imp.warn("%s: items inválido (esperado um array; campo ignorado)", label)
		}
		target.Items = models.JSONText(items)
	}
	if rec.Total.Set {
		total, err := parseImportDecimal(rec.Total.Value)
		if err != nil {
			imp.warn("%s: total inválido (campo ignorado)", label)
		}
		target.Total = total
	}

	switch {
	case isNew:
		imp.run.ServicesCreated++
	case snapshotService(target) != before:
		imp.run.ServicesUpdated++
	}

	// last_mileage do veículo só sobe.
	if vehicle != nil && target.Mileage != nil && (vehicle.row.LastMileage == nil || *target.Mileage > *vehicle.row.LastMileage) {
		km := *target.Mileage
		vehicle.row.LastMileage = &km
		vehicle.dirty = true
	}
}

// serviceVehicle acha o veículo da OS pelo vehicle_source_id; sem ele (ou não encontrado), pela placa.
func (imp *importer) serviceVehicle(label, vehicleSourceID, plate string) *trackedVehicle {
	if vehicleSourceID != "" {
		if tv := imp.cache.vehicleBySource[vehicleSourceID]; tv != nil {
			return tv
		}
		imp.warn("%s: %s não encontrado", label, referenceLabel("veículo", vehicleSourceID))
	}
	if plate == "" {
		return nil
	}
	return imp.cache.vehicleByPlate[plate]
}
