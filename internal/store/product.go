package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/dsig"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/gobl/uuid"

	guuid "github.com/google/uuid"

	"github.com/neo/goinvoice/internal/db/sqlc"
)

// SaveProduct upserts a product and replaces its child collections.
func (s *SQLite) SaveProduct(ctx context.Context, id string, it *org.Item) error {
	if it == nil {
		it = &org.Item{}
	}
	if it.UUID == "" {
		if id == "" {
			it.UUID = uuid.UUID(guuid.NewString())
		} else {
			it.UUID = uuid.UUID(id)
		}
	}
	if id == "" {
		id = it.UUID.String()
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.q.WithTx(tx)

	ext, err := jsonText(it.Ext)
	if err != nil {
		return fmt.Errorf("product ext: %w", err)
	}
	meta, err := jsonText(it.Meta)
	if err != nil {
		return fmt.Errorf("product meta: %w", err)
	}
	if err := q.UpsertProduct(ctx, sqlc.UpsertProductParams{
		ID:          id,
		Uuid:        nullStr(it.UUID.String()),
		Ref:         it.Ref.String(),
		Key:         nullStr(it.Key.String()),
		Name:        it.Name,
		Description: nullStr(it.Description),
		Currency:    it.Currency.String(),
		Price:       amountString(it.Price),
		List:        amountText(it.List),
		Discount:    amountText(it.Discount),
		Per:         amountText(it.Per),
		Unit:        nullStr(it.Unit.String()),
		Origin:      nullStr(it.Origin.String()),
		Ext:         ext,
		Meta:        meta,
	}); err != nil {
		return err
	}

	if err := q.DeleteProductIdentities(ctx, id); err != nil {
		return err
	}
	if err := q.DeleteProductAttributes(ctx, id); err != nil {
		return err
	}
	if err := q.DeleteProductImages(ctx, id); err != nil {
		return err
	}
	if err := q.DeleteProductAltPrices(ctx, id); err != nil {
		return err
	}

	for i, idn := range it.Identities {
		if idn == nil || idn.Code == "" {
			continue
		}
		idExt, err := jsonText(idn.Ext)
		if err != nil {
			return fmt.Errorf("product identity ext: %w", err)
		}
		if err := q.InsertProductIdentity(ctx, sqlc.InsertProductIdentityParams{
			ID: childID(idn.UUID), ProductID: id, Position: int64(i),
			Uuid: nullStr(idn.UUID.String()), Label: nullStr(idn.Label),
			Scope: nullStr(idn.Scope.String()), Country: nullStr(idn.Country.String()),
			Key: nullStr(idn.Key.String()), Type: nullStr(idn.Type.String()),
			Code: idn.Code.String(), Description: nullStr(idn.Description),
			Ext: idExt,
		}); err != nil {
			return err
		}
	}

	for i, attr := range it.Attributes {
		if attr == nil {
			continue
		}
		attrExt, err := jsonText(attr.Ext)
		if err != nil {
			return fmt.Errorf("product attribute ext: %w", err)
		}
		var date sql.NullString
		if attr.Date != nil {
			date = nullStr(attr.Date.String())
		}
		if err := q.InsertProductAttribute(ctx, sqlc.InsertProductAttributeParams{
			ID: guuid.NewString(), ProductID: id, Position: int64(i),
			Label: nullStr(attr.Label), Key: nullStr(attr.Key.String()),
			Type: nullStr(attr.Type.String()), Text: nullStr(attr.Text),
			Code: nullStr(attr.Code.String()), Amount: amountText(attr.Amount),
			Unit: nullStr(attr.Unit.String()), Date: date, Ext: attrExt,
		}); err != nil {
			return err
		}
	}

	for i, img := range it.Images {
		if img == nil {
			continue
		}
		digest, err := jsonText(img.Digest)
		if err != nil {
			return fmt.Errorf("product image digest: %w", err)
		}
		var meta sql.NullString
		if img.Meta != nil {
			meta, err = jsonText(*img.Meta)
			if err != nil {
				return fmt.Errorf("product image meta: %w", err)
			}
		}
		if err := q.InsertProductImage(ctx, sqlc.InsertProductImageParams{
			ID: childID(img.UUID), ProductID: id, Position: int64(i),
			Uuid: nullStr(img.UUID.String()), Label: nullStr(img.Label),
			Url: nullStr(img.URL), Data: img.Data, Mime: nullStr(img.MIME),
			Description: nullStr(img.Description), Alt: nullStr(img.Alt),
			Height: nullInt(int64(img.Height)), Width: nullInt(int64(img.Width)),
			Digest: digest, Meta: meta,
		}); err != nil {
			return err
		}
	}

	for i, alt := range it.AltPrices {
		if alt == nil || alt.Currency.String() == "" {
			continue
		}
		if err := q.InsertProductAltPrice(ctx, sqlc.InsertProductAltPriceParams{
			ID: guuid.NewString(), ProductID: id, Position: int64(i),
			Label: nullStr(alt.Label), Currency: alt.Currency.String(),
			Value: alt.Value.String(),
		}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetProductItem loads a product as a full GOBL org.Item.
func (s *SQLite) GetProductItem(ctx context.Context, id string) (*org.Item, error) {
	row, err := s.q.GetProduct(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrProductNotFound
		}
		return nil, err
	}
	it := &org.Item{
		Identify:    uuid.Identify{UUID: uuid.UUID(row.Uuid.String)},
		Ref:         cbc.Code(row.Ref),
		Key:         cbc.Key(row.Key.String),
		Name:        row.Name,
		Description: row.Description.String,
		Currency:    currency.Code(row.Currency),
		Price:       amountPtr(row.Price),
		List:        amountPtr(nullString(row.List)),
		Discount:    amountPtr(nullString(row.Discount)),
		Per:         amountPtr(nullString(row.Per)),
		Unit:        cbc.Key(row.Unit.String),
		Origin:      l10n.ISOCountryCode(row.Origin.String),
	}
	if err := decodeJSON(row.Ext, &it.Ext); err != nil {
		it.Ext = tax.Extensions{}
	}
	if err := decodeJSON(row.Meta, &it.Meta); err != nil {
		it.Meta = nil
	}

	ids, err := s.q.ListProductIdentities(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, r := range ids {
		it.Identities = append(it.Identities, productIdentityFromRow(r))
	}
	attrs, err := s.q.ListProductAttributes(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, r := range attrs {
		it.Attributes = append(it.Attributes, productAttributeFromRow(r))
	}
	imgs, err := s.q.ListProductImages(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, r := range imgs {
		it.Images = append(it.Images, productImageFromRow(r))
	}
	alts, err := s.q.ListProductAltPrices(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, r := range alts {
		it.AltPrices = append(it.AltPrices, &currency.Amount{
			Label:    r.Label.String,
			Currency: currency.Code(r.Currency),
			Value:    amountValue(r.Value),
		})
	}
	return it, nil
}

func nullString(s sql.NullString) string { return s.String }

func amountString(a *num.Amount) string {
	if a == nil {
		return ""
	}
	return a.String()
}

func amountText(a *num.Amount) sql.NullString {
	if a == nil {
		return sql.NullString{}
	}
	return nullStr(a.String())
}

func amountPtr(s string) *num.Amount {
	if s == "" {
		return nil
	}
	a, err := num.AmountFromString(s)
	if err != nil {
		return nil
	}
	return &a
}

func amountValue(s string) num.Amount {
	if a := amountPtr(s); a != nil {
		return *a
	}
	return num.Amount{}
}

func productIdentityFromRow(r sqlc.ProductIdentity) *org.Identity {
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

func productAttributeFromRow(r sqlc.ProductAttribute) *org.Attribute {
	a := &org.Attribute{
		Label:  r.Label.String,
		Key:    cbc.Key(r.Key.String),
		Type:   cbc.Code(r.Type.String),
		Text:   r.Text.String,
		Code:   cbc.Code(r.Code.String),
		Amount: amountPtr(r.Amount.String),
		Unit:   cbc.Key(r.Unit.String),
	}
	if r.Date.Valid && r.Date.String != "" {
		if t, err := time.Parse("2006-01-02", r.Date.String); err == nil {
			d := cal.DateOf(t)
			a.Date = &d
		}
	}
	_ = decodeJSON(r.Ext, &a.Ext)
	return a
}

func productImageFromRow(r sqlc.ProductImage) *org.Image {
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
