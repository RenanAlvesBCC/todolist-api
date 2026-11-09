package services

import (
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

func (imp *importer) customer(pos int, rec ImportCustomer) error {
	sourceID := rec.SourceID.trimmed()
	if sourceID == "" {
		imp.skip("cliente na posição %d sem source_id (ignorado)", pos)
		return nil
	}
	label := "cliente OnMotor " + sourceID

	target, err := imp.store.FindCustomerBySource(imp.wsID, models.SourceOnMotor, sourceID)
	if err != nil {
		return err
	}
	if target == nil && rec.CPF.Set {
		// Cliente local (sem source_id) com o mesmo CPF: vira o registro do OnMotor em vez de duplicar.
		if cpf, err := normalizeCPF(rec.CPF.Value); err == nil && cpf != "" {
			local, err := imp.store.FindCustomerByCPF(imp.wsID, cpf)
			if err != nil {
				return err
			}
			if local != nil && local.SourceID == nil {
				target = local
			}
		}
	}
	isNew := target == nil
	if isNew {
		if rec.Name.trimmed() == "" {
			imp.skip("%s sem nome (ignorado)", label)
			return nil
		}
		target = &models.Customer{WorkspaceID: imp.wsID}
	}
	before := snapshotCustomer(target)
	if !isNew && notNewer(rec.SourceUpdatedAt, target.SourceUpdatedAt) {
		return imp.customerVisitOnly(target, before, rec.LastVisitAt)
	}
	target.Source = models.SourceOnMotor
	target.SourceID = &sourceID

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
	if target.CPF != "" && target.CPF != before.cpf {
		holder, err := imp.store.FindCustomerByCPF(imp.wsID, target.CPF)
		if err != nil {
			return err
		}
		if holder != nil && holder.ID != target.ID {
			imp.warn("CPF duplicado em 2 clientes (%s, %s); mantido no primeiro", describeCustomer(holder), label)
			target.CPF = ""
		}
	}

	switch {
	case isNew:
		target.SyncedAt = &imp.now
		if err := imp.store.CreateCustomer(target); err != nil {
			return err
		}
		imp.run.CustomersCreated++
	case snapshotCustomer(target) != before:
		target.SyncedAt = &imp.now
		if err := imp.store.SaveCustomer(target); err != nil {
			return err
		}
		imp.run.CustomersUpdated++
	default:
		return imp.store.TouchCustomerSynced(target.ID, imp.now)
	}
	return nil
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
func (imp *importer) customerVisitOnly(c *models.Customer, before customerSnapshot, lastVisit FlexString) error {
	if lastVisit.Set {
		if t, err := parseImportTime(lastVisit.Value); err == nil && t != nil && (c.LastVisitAt == nil || t.After(*c.LastVisitAt)) {
			c.LastVisitAt = t
		}
	}
	if snapshotCustomer(c) == before {
		return imp.store.TouchCustomerSynced(c.ID, imp.now)
	}
	c.SyncedAt = &imp.now
	if err := imp.store.SaveCustomer(c); err != nil {
		return err
	}
	imp.run.CustomersUpdated++
	return nil
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

func (imp *importer) vehicle(pos int, rec ImportVehicle) error {
	sourceID := rec.SourceID.trimmed()
	if sourceID == "" {
		imp.skip("veículo na posição %d sem source_id (ignorado)", pos)
		return nil
	}
	label := "veículo OnMotor " + sourceID
	plate := normalizeCustomerPlate(rec.Plate.Value)

	target, err := imp.store.FindVehicleBySource(imp.wsID, models.SourceOnMotor, sourceID)
	if err != nil {
		return err
	}
	if target == nil && plate != "" {
		// Veículo local (sem source_id) com a mesma placa: vira o registro do OnMotor.
		local, err := imp.store.FindVehicleByPlate(imp.wsID, plate)
		if err != nil {
			return err
		}
		if local != nil && local.SourceID == nil {
			target = local
		}
	}
	isNew := target == nil
	if isNew {
		target = &models.CustomerVehicle{WorkspaceID: imp.wsID}
	}
	before := snapshotVehicle(target)
	target.Source = models.SourceOnMotor
	target.SourceID = &sourceID

	if ownerID := rec.CustomerSourceID.trimmed(); ownerID != "" {
		owner, err := imp.store.FindCustomerBySource(imp.wsID, models.SourceOnMotor, ownerID)
		if err != nil {
			return err
		}
		if owner != nil {
			target.CustomerID = &owner.ID
		} else {
			imp.warn("%s: cliente OnMotor %s não encontrado", label, ownerID)
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
	if target.Plate != "" && target.Plate != before.plate {
		holder, err := imp.store.FindVehicleByPlate(imp.wsID, target.Plate)
		if err != nil {
			return err
		}
		if holder != nil && holder.ID != target.ID {
			if err := imp.store.ClearVehiclePlate(holder.ID); err != nil {
				return err
			}
			imp.warn("placa duplicada: %s assumiu a placa do veículo %s", label, describeVehicle(holder))
		}
	}

	switch {
	case isNew:
		target.SyncedAt = &imp.now
		if err := imp.store.CreateVehicle(target); err != nil {
			return err
		}
		imp.run.VehiclesCreated++
	case snapshotVehicle(target) != before:
		target.SyncedAt = &imp.now
		if err := imp.store.SaveVehicle(target); err != nil {
			return err
		}
		imp.run.VehiclesUpdated++
	default:
		return imp.store.TouchVehicleSynced(target.ID, imp.now)
	}
	return nil
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

func (imp *importer) service(pos int, rec ImportService) error {
	sourceID := rec.SourceID.trimmed()
	if sourceID == "" {
		imp.skip("OS na posição %d sem source_id (ignorada)", pos)
		return nil
	}
	label := "OS OnMotor " + sourceID

	target, err := imp.store.FindServiceRecordBySource(imp.wsID, models.SourceOnMotor, sourceID)
	if err != nil {
		return err
	}
	isNew := target == nil
	if isNew {
		target = &models.ServiceRecord{WorkspaceID: imp.wsID, Source: models.SourceOnMotor, SourceID: sourceID, Items: "[]"}
	}
	before := snapshotService(target)

	if ownerID := rec.CustomerSourceID.trimmed(); ownerID != "" {
		owner, err := imp.store.FindCustomerBySource(imp.wsID, models.SourceOnMotor, ownerID)
		if err != nil {
			return err
		}
		if owner != nil {
			target.CustomerID = &owner.ID
		} else {
			imp.warn("%s: cliente OnMotor %s não encontrado", label, ownerID)
		}
	}

	plate := normalizeCustomerPlate(rec.Plate.Value)
	vehicle, err := imp.serviceVehicle(label, rec.VehicleSourceID.trimmed(), plate)
	if err != nil {
		return err
	}
	if vehicle != nil {
		target.CustomerVehicleID = &vehicle.ID
		if target.CustomerID == nil && vehicle.CustomerID != nil {
			id := *vehicle.CustomerID
			target.CustomerID = &id
		}
	}
	switch {
	case rec.Plate.Set:
		target.Plate = plate
	case target.Plate == "" && vehicle != nil:
		target.Plate = vehicle.Plate
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
		target.SyncedAt = &imp.now
		if err := imp.store.CreateServiceRecord(target); err != nil {
			return err
		}
		imp.run.ServicesCreated++
	case snapshotService(target) != before:
		target.SyncedAt = &imp.now
		if err := imp.store.SaveServiceRecord(target); err != nil {
			return err
		}
		imp.run.ServicesUpdated++
	default:
		if err := imp.store.TouchServiceRecordSynced(target.ID, imp.now); err != nil {
			return err
		}
	}

	if target.CustomerVehicleID != nil && target.Mileage != nil {
		return imp.store.RaiseVehicleMileage(*target.CustomerVehicleID, *target.Mileage)
	}
	return nil
}

// serviceVehicle acha o veículo da OS pelo vehicle_source_id; sem ele (ou não encontrado), pela placa.
func (imp *importer) serviceVehicle(label, vehicleSourceID, plate string) (*models.CustomerVehicle, error) {
	if vehicleSourceID != "" {
		v, err := imp.store.FindVehicleBySource(imp.wsID, models.SourceOnMotor, vehicleSourceID)
		if err != nil || v != nil {
			return v, err
		}
		imp.warn("%s: veículo OnMotor %s não encontrado", label, vehicleSourceID)
	}
	if plate == "" {
		return nil, nil
	}
	return imp.store.FindVehicleByPlate(imp.wsID, plate)
}
