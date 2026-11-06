package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/dsig"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/gobl/uuid"

	guuid "github.com/google/uuid"

	"github.com/neo/goinvoice/internal/db/sqlc"
)

// PartyKind distinguishes the two catalog party tables.
type PartyKind string

const (
	PartyCustomer PartyKind = "customer"
	PartySupplier PartyKind = "supplier"
)

// SaveCustomerParty upserts a customer and replaces its child collections.
func (s *SQLite) SaveCustomerParty(ctx context.Context, id string, p *org.Party) error {
	return s.saveParty(ctx, PartyCustomer, id, p)
}

// SaveSupplierParty upserts a supplier and replaces its child collections.
func (s *SQLite) SaveSupplierParty(ctx context.Context, id string, p *org.Party) error {
	return s.saveParty(ctx, PartySupplier, id, p)
}

// GetCustomerParty loads a customer as a full GOBL party.
func (s *SQLite) GetCustomerParty(ctx context.Context, id string) (*org.Party, error) {
	row, err := s.q.GetCustomer(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrCustomerNotFound
		}
		return nil, err
	}
	p := partyFromRow(row)
	if err := s.loadPartyChildren(ctx, PartyCustomer, id, p); err != nil {
		return nil, err
	}
	return p, nil
}

// GetSupplierParty loads a supplier as a full GOBL party.
func (s *SQLite) GetSupplierParty(ctx context.Context, id string) (*org.Party, error) {
	row, err := s.q.GetSupplier(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrSupplierNotFound
		}
		return nil, err
	}
	p := partyFromRow(row)
	if err := s.loadPartyChildren(ctx, PartySupplier, id, p); err != nil {
		return nil, err
	}
	return p, nil
}

// saveParty writes the scalar party row and syncs every child collection in a
// single transaction. Child rows are replaced wholesale, which keeps ordering
// and removal semantics simple.
func (s *SQLite) saveParty(ctx context.Context, kind PartyKind, id string, p *org.Party) error {
	if p == nil {
		p = &org.Party{}
	}
	if p.UUID == "" {
		if id == "" {
			p.UUID = uuid.UUID(guuid.NewString())
		} else {
			p.UUID = uuid.UUID(id)
		}
	}
	if id == "" {
		id = p.UUID.String()
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.q.WithTx(tx)

	scalars, err := scalarsFromParty(id, p)
	if err != nil {
		return err
	}
	// agent_id is a self-FK: keep it only when it points at an existing party,
	// otherwise the agent reference would violate the constraint.
	if scalars.AgentID.Valid {
		table := "customers"
		if kind == PartySupplier {
			table = "suppliers"
		}
		var n int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(1) FROM "+table+" WHERE id = ?", scalars.AgentID.String).Scan(&n); err != nil {
			return err
		}
		if n == 0 {
			scalars.AgentID = sql.NullString{}
		}
	}
	if kind == PartyCustomer {
		if err := q.UpsertCustomer(ctx, scalars.customer()); err != nil {
			return err
		}
	} else {
		if err := q.UpsertSupplier(ctx, scalars.supplier()); err != nil {
			return err
		}
	}

	if err := s.saveChildren(ctx, q, kind, id, p); err != nil {
		return err
	}
	return tx.Commit()
}

// customer/supplier scalars shared by both parent tables.
type partyScalars struct {
	ID          string
	Uuid        sql.NullString
	Regime      sql.NullString
	Label       sql.NullString
	Name        string
	Alias       sql.NullString
	TaxCountry  sql.NullString
	TaxCode     sql.NullString
	TaxScheme   sql.NullString
	TaxType     sql.NullString
	AgentID     sql.NullString
	Email       sql.NullString
	AddressLine sql.NullString
	Locality    sql.NullString
	PostCode    sql.NullString
	Country     sql.NullString
	Ext         sql.NullString
	Meta        sql.NullString
}

func (s partyScalars) customer() sqlc.UpsertCustomerParams {
	return sqlc.UpsertCustomerParams{
		ID: s.ID, Uuid: s.Uuid, Regime: s.Regime, Label: s.Label, Name: s.Name,
		Alias: s.Alias, TaxCountry: s.TaxCountry, TaxCode: s.TaxCode,
		TaxScheme: s.TaxScheme, TaxType: s.TaxType, AgentID: s.AgentID,
		Email: s.Email, AddressLine: s.AddressLine, Locality: s.Locality,
		PostCode: s.PostCode, Country: s.Country, Ext: s.Ext, Meta: s.Meta,
	}
}

func (s partyScalars) supplier() sqlc.UpsertSupplierParams {
	return sqlc.UpsertSupplierParams{
		ID: s.ID, Uuid: s.Uuid, Regime: s.Regime, Label: s.Label, Name: s.Name,
		Alias: s.Alias, TaxCountry: s.TaxCountry, TaxCode: s.TaxCode,
		TaxScheme: s.TaxScheme, TaxType: s.TaxType, AgentID: s.AgentID,
		Email: s.Email, AddressLine: s.AddressLine, Locality: s.Locality,
		PostCode: s.PostCode, Country: s.Country, Ext: s.Ext, Meta: s.Meta,
	}
}

func scalarsFromParty(id string, p *org.Party) (partyScalars, error) {
	s := partyScalars{
		ID:     id,
		Uuid:   nullStr(p.UUID.String()),
		Regime: nullStr(string(p.Country)),
		Label:  nullStr(p.Label),
		Name:   p.Name,
		Alias:  nullStr(p.Alias),
	}
	if p.TaxID != nil {
		s.TaxCountry = nullStr(p.TaxID.Country.String())
		s.TaxCode = nullStr(p.TaxID.Code.String())
		s.TaxScheme = nullStr(p.TaxID.Scheme.String())
		s.TaxType = nullStr(p.TaxID.Type.String())
	}
	if p.Agent != nil {
		s.AgentID = nullStr(p.Agent.UUID.String())
	}
	// Denormalized first email/address keep the simple list/edit UI working
	// without joins; child tables remain the source of truth.
	if len(p.Emails) > 0 && p.Emails[0] != nil {
		s.Email = nullStr(p.Emails[0].Address)
	}
	if len(p.Addresses) > 0 && p.Addresses[0] != nil {
		a := p.Addresses[0]
		s.AddressLine = nullStr(a.LineOne())
		s.Locality = nullStr(a.Locality)
		s.PostCode = nullStr(a.Code.String())
		s.Country = nullStr(a.Country.String())
	}
	ext, err := jsonText(p.Ext)
	if err != nil {
		return s, fmt.Errorf("party ext: %w", err)
	}
	s.Ext = ext
	meta, err := jsonText(p.Meta)
	if err != nil {
		return s, fmt.Errorf("party meta: %w", err)
	}
	s.Meta = meta
	return s, nil
}

func nullStr(v string) sql.NullString {
	if v == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: v, Valid: true}
}

func jsonText(v any) (sql.NullString, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return sql.NullString{}, err
	}
	if string(raw) == "{}" || string(raw) == "null" || string(raw) == `""` {
		return sql.NullString{}, nil
	}
	return sql.NullString{String: string(raw), Valid: true}, nil
}

func decodeJSON(raw sql.NullString, dst any) error {
	if !raw.Valid || raw.String == "" {
		return nil
	}
	return json.Unmarshal([]byte(raw.String), dst)
}

// partyFromRow builds the scalar portion shared by customers and suppliers.
func partyFromRow(row interface{}) *org.Party {
	var (
		uuidVal, regime, label, name, alias     sql.NullString
		taxCountry, taxCode, taxScheme, taxType sql.NullString
		agentID, ext, meta                      sql.NullString
	)
	switch r := row.(type) {
	case sqlc.Customer:
		uuidVal, regime, label, name, alias = r.Uuid, r.Regime, r.Label, nullStr(r.Name), r.Alias
		taxCountry, taxCode, taxScheme, taxType = r.TaxCountry, r.TaxCode, r.TaxScheme, r.TaxType
		agentID, ext, meta = r.AgentID, r.Ext, r.Meta
	case sqlc.Supplier:
		uuidVal, regime, label, name, alias = r.Uuid, r.Regime, r.Label, nullStr(r.Name), r.Alias
		taxCountry, taxCode, taxScheme, taxType = r.TaxCountry, r.TaxCode, r.TaxScheme, r.TaxType
		agentID, ext, meta = r.AgentID, r.Ext, r.Meta
	}

	p := &org.Party{
		Regime:   tax.Regime{Country: tax.RegimeCode(regime.String)},
		Identify: uuid.Identify{UUID: uuid.UUID(uuidVal.String)},
		Label:    label.String,
		Name:     name.String,
		Alias:    alias.String,
	}
	if taxCountry.Valid || taxCode.Valid || taxScheme.Valid || taxType.Valid {
		p.TaxID = &tax.Identity{
			Country: l10n.TaxCountryCode(taxCountry.String),
			Code:    cbc.Code(taxCode.String),
			Scheme:  cbc.Code(taxScheme.String),
			Type:    cbc.Key(taxType.String),
		}
	}
	if agentID.Valid && agentID.String != "" {
		p.Agent = &org.Party{Identify: uuid.Identify{UUID: uuid.UUID(agentID.String)}}
	}
	if err := decodeJSON(ext, &p.Ext); err != nil {
		p.Ext = tax.Extensions{}
	}
	if err := decodeJSON(meta, &p.Meta); err != nil {
		p.Meta = nil
	}
	return p
}

// loadPartyChildren fills every repeated party collection from the child tables.
func (s *SQLite) loadPartyChildren(ctx context.Context, kind PartyKind, id string, p *org.Party) error {
	// Each collection is loaded with its concrete query; the switch keeps the
	// generated types explicit without reflection.
	if kind == PartyCustomer {
		arg := nullStr(id)
		addrs, err := s.q.ListPartyAddressesByCustomer(ctx, arg)
		if err != nil {
			return err
		}
		for _, r := range addrs {
			p.Addresses = append(p.Addresses, addressFromRow(r))
		}
		emails, err := s.q.ListPartyEmailsByCustomer(ctx, arg)
		if err != nil {
			return err
		}
		for _, r := range emails {
			p.Emails = append(p.Emails, emailFromRow(r))
		}
		tels, err := s.q.ListPartyTelephonesByCustomer(ctx, arg)
		if err != nil {
			return err
		}
		for _, r := range tels {
			p.Telephones = append(p.Telephones, telephoneFromRow(r))
		}
		sites, err := s.q.ListPartyWebsitesByCustomer(ctx, arg)
		if err != nil {
			return err
		}
		for _, r := range sites {
			p.Websites = append(p.Websites, websiteFromRow(r))
		}
		ids, err := s.q.ListPartyIdentitiesByCustomer(ctx, arg)
		if err != nil {
			return err
		}
		for _, r := range ids {
			p.Identities = append(p.Identities, identityFromRow(r))
		}
		eps, err := s.q.ListPartyEndpointsByCustomer(ctx, arg)
		if err != nil {
			return err
		}
		for _, r := range eps {
			p.Endpoints = append(p.Endpoints, endpointFromRow(r))
		}
		inboxes, err := s.q.ListPartyInboxesByCustomer(ctx, arg)
		if err != nil {
			return err
		}
		for _, r := range inboxes {
			p.Inboxes = append(p.Inboxes, inboxFromRow(r))
		}
		logos, err := s.q.ListPartyLogosByCustomer(ctx, arg)
		if err != nil {
			return err
		}
		for _, r := range logos {
			p.Logos = append(p.Logos, logoFromRow(r))
		}
		people, err := s.q.ListPartyPeopleByCustomer(ctx, arg)
		if err != nil {
			return err
		}
		for _, r := range people {
			if person, err := personFromRow(r); err == nil {
				p.People = append(p.People, person)
			}
		}
		return nil
	}

	arg := nullStr(id)
	addrs, err := s.q.ListPartyAddressesBySupplier(ctx, arg)
	if err != nil {
		return err
	}
	for _, r := range addrs {
		p.Addresses = append(p.Addresses, addressFromRow(r))
	}
	emails, err := s.q.ListPartyEmailsBySupplier(ctx, arg)
	if err != nil {
		return err
	}
	for _, r := range emails {
		p.Emails = append(p.Emails, emailFromRow(r))
	}
	tels, err := s.q.ListPartyTelephonesBySupplier(ctx, arg)
	if err != nil {
		return err
	}
	for _, r := range tels {
		p.Telephones = append(p.Telephones, telephoneFromRow(r))
	}
	sites, err := s.q.ListPartyWebsitesBySupplier(ctx, arg)
	if err != nil {
		return err
	}
	for _, r := range sites {
		p.Websites = append(p.Websites, websiteFromRow(r))
	}
	ids, err := s.q.ListPartyIdentitiesBySupplier(ctx, arg)
	if err != nil {
		return err
	}
	for _, r := range ids {
		p.Identities = append(p.Identities, identityFromRow(r))
	}
	eps, err := s.q.ListPartyEndpointsBySupplier(ctx, arg)
	if err != nil {
		return err
	}
	for _, r := range eps {
		p.Endpoints = append(p.Endpoints, endpointFromRow(r))
	}
	inboxes, err := s.q.ListPartyInboxesBySupplier(ctx, arg)
	if err != nil {
		return err
	}
	for _, r := range inboxes {
		p.Inboxes = append(p.Inboxes, inboxFromRow(r))
	}
	logos, err := s.q.ListPartyLogosBySupplier(ctx, arg)
	if err != nil {
		return err
	}
	for _, r := range logos {
		p.Logos = append(p.Logos, logoFromRow(r))
	}
	people, err := s.q.ListPartyPeopleBySupplier(ctx, arg)
	if err != nil {
		return err
	}
	for _, r := range people {
		if person, err := personFromRow(r); err == nil {
			p.People = append(p.People, person)
		}
	}
	return nil
}

func addressFromRow(r sqlc.PartyAddress) *org.Address {
	a := &org.Address{
		Identify:      uuid.Identify{UUID: uuid.UUID(r.Uuid.String)},
		Label:         r.Label.String,
		PostOfficeBox: r.PoBox.String,
		Number:        r.Num.String,
		Floor:         r.Floor.String,
		Block:         r.Block.String,
		Door:          r.Door.String,
		Street:        r.Street.String,
		StreetExtra:   r.StreetExtra.String,
		Locality:      r.Locality.String,
		Region:        r.Region.String,
		State:         cbc.Code(r.State.String),
		Code:          cbc.Code(r.Code.String),
		Country:       l10n.ISOCountryCode(r.Country.String),
	}
	if r.Coords.Valid && r.Coords.String != "" {
		var c org.Coordinates
		if err := json.Unmarshal([]byte(r.Coords.String), &c); err == nil {
			a.Coordinates = &c
		}
	}
	_ = decodeJSON(r.Meta, &a.Meta)
	return a
}

func emailFromRow(r sqlc.PartyEmail) *org.Email {
	e := &org.Email{
		Identify: uuid.Identify{UUID: uuid.UUID(r.Uuid.String)},
		Label:    r.Label.String,
		Address:  r.Addr,
	}
	_ = decodeJSON(r.Meta, &e.Meta)
	return e
}

func telephoneFromRow(r sqlc.PartyTelephone) *org.Telephone {
	return &org.Telephone{
		Identify: uuid.Identify{UUID: uuid.UUID(r.Uuid.String)},
		Label:    r.Label.String,
		Number:   r.Num,
	}
}

func websiteFromRow(r sqlc.PartyWebsite) *org.Website {
	return &org.Website{
		Identify: uuid.Identify{UUID: uuid.UUID(r.Uuid.String)},
		Label:    r.Label.String,
		Title:    r.Title.String,
		URL:      r.Url,
	}
}

func identityFromRow(r sqlc.PartyIdentity) *org.Identity {
	i := &org.Identity{
		Identify:    uuid.Identify{UUID: uuid.UUID(r.Uuid.String)},
		Label:       r.Label.String,
		Scope:       cbc.Key(r.Scope.String),
		Country:     l10n.ISOCountryCode(r.Country.String),
		Key:         cbc.Key(r.Key.String),
		Type:        cbc.Code(r.Type.String),
		Code:        cbc.Code(r.Code),
		Description: r.Description.String,
	}
	_ = decodeJSON(r.Ext, &i.Ext)
	return i
}

func endpointFromRow(r sqlc.PartyEndpoint) *org.Endpoint {
	return &org.Endpoint{
		Identify: uuid.Identify{UUID: uuid.UUID(r.Uuid.String)},
		Label:    r.Label.String,
		URI:      cbc.URI(r.Uri),
	}
}

func inboxFromRow(r sqlc.PartyInbox) *org.Inbox {
	return &org.Inbox{
		Identify: uuid.Identify{UUID: uuid.UUID(r.Uuid.String)},
		Label:    r.Label.String,
		Key:      cbc.Key(r.Key.String),
		Scheme:   cbc.Code(r.Scheme.String),
		Code:     cbc.Code(r.Code.String),
		URL:      r.Url.String,
		Email:    r.Email.String,
	}
}

func logoFromRow(r sqlc.PartyLogo) *org.Image {
	img := &org.Image{
		Identify:    uuid.Identify{UUID: uuid.UUID(r.Uuid.String)},
		Label:       r.Label.String,
		URL:         r.Url.String,
		Data:        r.Data,
		MIME:        r.Mime.String,
		Description: r.Description.String,
		Alt:         r.Alt.String,
		Height:      int32(r.Height.Int64),
		Width:       int32(r.Width.Int64),
	}
	if r.Digest.Valid && r.Digest.String != "" {
		var d dsig.Digest
		if err := json.Unmarshal([]byte(r.Digest.String), &d); err == nil {
			img.Digest = &d
		}
	}
	if r.Meta.Valid && r.Meta.String != "" {
		var m cbc.Meta
		if err := json.Unmarshal([]byte(r.Meta.String), &m); err == nil {
			img.Meta = &m
		}
	}
	return img
}

type personJSON struct {
	Identities []*org.Identity  `json:"identities,omitempty"`
	Addresses  []*org.Address   `json:"addresses,omitempty"`
	Emails     []*org.Email     `json:"emails,omitempty"`
	Telephones []*org.Telephone `json:"telephones,omitempty"`
	Avatars    []*org.Image     `json:"avatars,omitempty"`
}

func personFromRow(r sqlc.PartyPerson) (*org.Person, error) {
	p := &org.Person{
		Identify: uuid.Identify{UUID: uuid.UUID(r.Uuid.String)},
		Label:    r.Label.String,
		Key:      cbc.Key(r.Key.String),
		Role:     r.Role.String,
		Name: &org.Name{
			Identify: uuid.Identify{UUID: uuid.UUID(r.NameUuid.String)},
			Alias:    r.NameAlias.String,
			Prefix:   r.NamePrefix.String,
			Given:    r.NameGiven.String,
			Middle:   r.NameMiddle.String,
			Surname:  r.NameSurname.String,
			Surname2: r.NameSurname2.String,
			Suffix:   r.NameSuffix.String,
		},
	}
	_ = decodeJSON(r.NameMeta, &p.Name.Meta)
	_ = decodeJSON(r.Meta, &p.Meta)
	var nested personJSON
	if err := decodeJSON(r.Identities, &nested.Identities); err != nil {
		return nil, err
	}
	if err := decodeJSON(r.Addresses, &nested.Addresses); err != nil {
		return nil, err
	}
	if err := decodeJSON(r.Emails, &nested.Emails); err != nil {
		return nil, err
	}
	if err := decodeJSON(r.Telephones, &nested.Telephones); err != nil {
		return nil, err
	}
	if err := decodeJSON(r.Avatars, &nested.Avatars); err != nil {
		return nil, err
	}
	p.Identities = nested.Identities
	p.Addresses = nested.Addresses
	p.Emails = nested.Emails
	p.Telephones = nested.Telephones
	p.Avatars = nested.Avatars
	return p, nil
}
