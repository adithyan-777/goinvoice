package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/uuid"

	guuid "github.com/google/uuid"

	"github.com/neo/goinvoice/internal/db/sqlc"
)

func parentIDs(kind PartyKind, id string) (sql.NullString, sql.NullString) {
	if kind == PartyCustomer {
		return nullStr(id), sql.NullString{}
	}
	return sql.NullString{}, nullStr(id)
}

func childID(id uuid.UUID) string {
	if id != "" {
		return id.String()
	}
	return guuid.NewString()
}

// saveChildren replaces every repeated party collection inside the open tx.
func (s *SQLite) saveChildren(ctx context.Context, q *sqlc.Queries, kind PartyKind, id string, p *org.Party) error {
	customerID, supplierID := parentIDs(kind, id)

	if kind == PartyCustomer {
		if err := q.DeletePartyAddressesByCustomer(ctx, customerID); err != nil {
			return err
		}
		if err := q.DeletePartyEmailsByCustomer(ctx, customerID); err != nil {
			return err
		}
		if err := q.DeletePartyTelephonesByCustomer(ctx, customerID); err != nil {
			return err
		}
		if err := q.DeletePartyWebsitesByCustomer(ctx, customerID); err != nil {
			return err
		}
		if err := q.DeletePartyIdentitiesByCustomer(ctx, customerID); err != nil {
			return err
		}
		if err := q.DeletePartyEndpointsByCustomer(ctx, customerID); err != nil {
			return err
		}
		if err := q.DeletePartyInboxesByCustomer(ctx, customerID); err != nil {
			return err
		}
		if err := q.DeletePartyLogosByCustomer(ctx, customerID); err != nil {
			return err
		}
		if err := q.DeletePartyPeopleByCustomer(ctx, customerID); err != nil {
			return err
		}
	} else {
		if err := q.DeletePartyAddressesBySupplier(ctx, supplierID); err != nil {
			return err
		}
		if err := q.DeletePartyEmailsBySupplier(ctx, supplierID); err != nil {
			return err
		}
		if err := q.DeletePartyTelephonesBySupplier(ctx, supplierID); err != nil {
			return err
		}
		if err := q.DeletePartyWebsitesBySupplier(ctx, supplierID); err != nil {
			return err
		}
		if err := q.DeletePartyIdentitiesBySupplier(ctx, supplierID); err != nil {
			return err
		}
		if err := q.DeletePartyEndpointsBySupplier(ctx, supplierID); err != nil {
			return err
		}
		if err := q.DeletePartyInboxesBySupplier(ctx, supplierID); err != nil {
			return err
		}
		if err := q.DeletePartyLogosBySupplier(ctx, supplierID); err != nil {
			return err
		}
		if err := q.DeletePartyPeopleBySupplier(ctx, supplierID); err != nil {
			return err
		}
	}

	for i, a := range p.Addresses {
		if a == nil {
			continue
		}
		coords, err := jsonText(a.Coordinates)
		if err != nil {
			return fmt.Errorf("address coords: %w", err)
		}
		meta, err := jsonText(a.Meta)
		if err != nil {
			return fmt.Errorf("address meta: %w", err)
		}
		if err := q.InsertPartyAddress(ctx, sqlc.InsertPartyAddressParams{
			ID: childID(a.UUID), CustomerID: customerID, SupplierID: supplierID,
			Position: int64(i), Uuid: nullStr(a.UUID.String()), Label: nullStr(a.Label),
			PoBox: nullStr(a.PostOfficeBox), Num: nullStr(a.Number), Floor: nullStr(a.Floor),
			Block: nullStr(a.Block), Door: nullStr(a.Door), Street: nullStr(a.Street),
			StreetExtra: nullStr(a.StreetExtra), Locality: nullStr(a.Locality),
			Region: nullStr(a.Region), State: nullStr(a.State.String()),
			Code: nullStr(a.Code.String()), Country: nullStr(a.Country.String()),
			Coords: coords, Meta: meta,
		}); err != nil {
			return err
		}
	}

	for i, e := range p.Emails {
		if e == nil || e.Address == "" {
			continue
		}
		meta, err := jsonText(e.Meta)
		if err != nil {
			return fmt.Errorf("email meta: %w", err)
		}
		if err := q.InsertPartyEmail(ctx, sqlc.InsertPartyEmailParams{
			ID: childID(e.UUID), CustomerID: customerID, SupplierID: supplierID,
			Position: int64(i), Uuid: nullStr(e.UUID.String()), Label: nullStr(e.Label),
			Addr: e.Address, Meta: meta,
		}); err != nil {
			return err
		}
	}

	for i, t := range p.Telephones {
		if t == nil || t.Number == "" {
			continue
		}
		if err := q.InsertPartyTelephone(ctx, sqlc.InsertPartyTelephoneParams{
			ID: childID(t.UUID), CustomerID: customerID, SupplierID: supplierID,
			Position: int64(i), Uuid: nullStr(t.UUID.String()), Label: nullStr(t.Label),
			Num: t.Number,
		}); err != nil {
			return err
		}
	}

	for i, w := range p.Websites {
		if w == nil || w.URL == "" {
			continue
		}
		if err := q.InsertPartyWebsite(ctx, sqlc.InsertPartyWebsiteParams{
			ID: childID(w.UUID), CustomerID: customerID, SupplierID: supplierID,
			Position: int64(i), Uuid: nullStr(w.UUID.String()), Label: nullStr(w.Label),
			Title: nullStr(w.Title), Url: w.URL,
		}); err != nil {
			return err
		}
	}

	for i, idn := range p.Identities {
		if idn == nil || idn.Code == "" {
			continue
		}
		ext, err := jsonText(idn.Ext)
		if err != nil {
			return fmt.Errorf("identity ext: %w", err)
		}
		if err := q.InsertPartyIdentity(ctx, sqlc.InsertPartyIdentityParams{
			ID: childID(idn.UUID), CustomerID: customerID, SupplierID: supplierID,
			Position: int64(i), Uuid: nullStr(idn.UUID.String()), Label: nullStr(idn.Label),
			Scope: nullStr(idn.Scope.String()), Country: nullStr(idn.Country.String()),
			Key: nullStr(idn.Key.String()), Type: nullStr(idn.Type.String()),
			Code: idn.Code.String(), Description: nullStr(idn.Description), Ext: ext,
		}); err != nil {
			return err
		}
	}

	for i, ep := range p.Endpoints {
		if ep == nil || ep.URI == "" {
			continue
		}
		if err := q.InsertPartyEndpoint(ctx, sqlc.InsertPartyEndpointParams{
			ID: childID(ep.UUID), CustomerID: customerID, SupplierID: supplierID,
			Position: int64(i), Uuid: nullStr(ep.UUID.String()), Label: nullStr(ep.Label),
			Uri: ep.URI.String(),
		}); err != nil {
			return err
		}
	}

	for i, in := range p.Inboxes {
		if in == nil {
			continue
		}
		if err := q.InsertPartyInbox(ctx, sqlc.InsertPartyInboxParams{
			ID: childID(in.UUID), CustomerID: customerID, SupplierID: supplierID,
			Position: int64(i), Uuid: nullStr(in.UUID.String()), Label: nullStr(in.Label),
			Key: nullStr(in.Key.String()), Scheme: nullStr(in.Scheme.String()),
			Code: nullStr(in.Code.String()), Url: nullStr(in.URL), Email: nullStr(in.Email),
		}); err != nil {
			return err
		}
	}

	for i, img := range p.Logos {
		if img == nil {
			continue
		}
		digest, err := jsonText(img.Digest)
		if err != nil {
			return fmt.Errorf("logo digest: %w", err)
		}
		var meta sql.NullString
		if img.Meta != nil {
			meta, err = jsonText(*img.Meta)
			if err != nil {
				return fmt.Errorf("logo meta: %w", err)
			}
		}
		if err := q.InsertPartyLogo(ctx, sqlc.InsertPartyLogoParams{
			ID: childID(img.UUID), CustomerID: customerID, SupplierID: supplierID,
			Position: int64(i), Uuid: nullStr(img.UUID.String()), Label: nullStr(img.Label),
			Url: nullStr(img.URL), Data: img.Data, Mime: nullStr(img.MIME),
			Description: nullStr(img.Description), Alt: nullStr(img.Alt),
			Height: nullInt(int64(img.Height)), Width: nullInt(int64(img.Width)),
			Digest: digest, Meta: meta,
		}); err != nil {
			return err
		}
	}

	for i, person := range p.People {
		if person == nil {
			continue
		}
		if err := insertPerson(ctx, q, customerID, supplierID, int64(i), person); err != nil {
			return err
		}
	}
	return nil
}

func insertPerson(ctx context.Context, q *sqlc.Queries, customerID, supplierID sql.NullString, pos int64, p *org.Person) error {
	name := p.Name
	if name == nil {
		name = &org.Name{}
	}
	nameMeta, err := jsonText(name.Meta)
	if err != nil {
		return fmt.Errorf("person name meta: %w", err)
	}
	identities, err := jsonText(p.Identities)
	if err != nil {
		return fmt.Errorf("person identities: %w", err)
	}
	addresses, err := jsonText(p.Addresses)
	if err != nil {
		return fmt.Errorf("person addresses: %w", err)
	}
	emails, err := jsonText(p.Emails)
	if err != nil {
		return fmt.Errorf("person emails: %w", err)
	}
	telephones, err := jsonText(p.Telephones)
	if err != nil {
		return fmt.Errorf("person telephones: %w", err)
	}
	avatars, err := jsonText(p.Avatars)
	if err != nil {
		return fmt.Errorf("person avatars: %w", err)
	}
	meta, err := jsonText(p.Meta)
	if err != nil {
		return fmt.Errorf("person meta: %w", err)
	}
	return q.InsertPartyPerson(ctx, sqlc.InsertPartyPersonParams{
		ID: childID(p.UUID), CustomerID: customerID, SupplierID: supplierID,
		Position: pos, Uuid: nullStr(p.UUID.String()), Label: nullStr(p.Label),
		Key: nullStr(p.Key.String()), Role: nullStr(p.Role),
		NameUuid: nullStr(name.UUID.String()), NameAlias: nullStr(name.Alias),
		NamePrefix: nullStr(name.Prefix), NameGiven: nullStr(name.Given),
		NameMiddle: nullStr(name.Middle), NameSurname: nullStr(name.Surname),
		NameSurname2: nullStr(name.Surname2), NameSuffix: nullStr(name.Suffix),
		NameMeta: nameMeta, Identities: identities, Addresses: addresses,
		Emails: emails, Telephones: telephones, Avatars: avatars, Meta: meta,
	})
}

func nullInt(v int64) sql.NullInt64 {
	if v == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: v, Valid: true}
}
