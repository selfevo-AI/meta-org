package costing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrValidation = errors.New("validation")
	ErrNotFound   = errors.New("not found")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) UpsertCurrency(ctx context.Context, input CreateCurrencyInput) (*Currency, error) {
	input.Code = normalizeCurrency(input.Code)
	if input.Code == "" {
		return nil, fmt.Errorf("%w: code is required", ErrValidation)
	}
	if input.CurrencyType == "" {
		input.CurrencyType = CurrencyFiat
	}
	if input.PrecisionDigits <= 0 {
		if input.CurrencyType == CurrencyVirtual {
			input.PrecisionDigits = 8
		} else {
			input.PrecisionDigits = 2
		}
	}
	return s.repo.UpsertCurrency(ctx, input)
}

func (s *Service) ListCurrencies(ctx context.Context) ([]Currency, error) {
	return s.repo.ListCurrencies(ctx)
}

func (s *Service) CreateExchangeRate(ctx context.Context, input CreateExchangeRateInput) (*ExchangeRateVersion, error) {
	input.FromCurrency = normalizeCurrency(input.FromCurrency)
	input.ToCurrency = normalizeCurrency(input.ToCurrency)
	if input.FromCurrency == "" || input.ToCurrency == "" {
		return nil, fmt.Errorf("%w: from_currency and to_currency are required", ErrValidation)
	}
	if input.FromCurrency == input.ToCurrency {
		return nil, fmt.Errorf("%w: from_currency and to_currency must differ", ErrValidation)
	}
	if input.Rate <= 0 {
		return nil, fmt.Errorf("%w: rate must be positive", ErrValidation)
	}
	if input.Source == "" {
		input.Source = RateSourceManual
	}
	return s.repo.CreateExchangeRate(ctx, input)
}

func (s *Service) ListExchangeRates(ctx context.Context, limit int) ([]ExchangeRateVersion, error) {
	return s.repo.ListExchangeRates(ctx, limit)
}

func (s *Service) Convert(ctx context.Context, input ConvertInput) (*ConversionResult, error) {
	input.FromCurrency = normalizeCurrency(input.FromCurrency)
	input.ToCurrency = normalizeCurrency(input.ToCurrency)
	if input.FromCurrency == "" {
		input.FromCurrency = BaseCurrency
	}
	if input.ToCurrency == "" {
		input.ToCurrency = BaseCurrency
	}
	at := time.Now().UTC()
	if input.At != nil {
		at = *input.At
	}
	result, err := s.convert(ctx, input.Amount, input.FromCurrency, input.ToCurrency, at)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Service) CreateRateCard(ctx context.Context, input CreateRateCardInput) (*CostRateCard, error) {
	input.SubjectType = strings.TrimSpace(input.SubjectType)
	if input.SubjectType == "" {
		return nil, fmt.Errorf("%w: subject_type is required", ErrValidation)
	}
	if input.RateType == "" {
		input.RateType = "fixed"
	}
	if input.Status == "" {
		input.Status = "active"
	}
	input.Currency = defaultCurrency(input.Currency)
	conversion, err := s.convert(ctx, input.Amount, input.Currency, BaseCurrency, effectiveTime(input.EffectiveFrom))
	if err != nil {
		return nil, err
	}
	return s.repo.CreateRateCard(ctx, input, conversion)
}

func (s *Service) ListRateCards(ctx context.Context, limit int) ([]CostRateCard, error) {
	return s.repo.ListRateCards(ctx, limit)
}

func (s *Service) CreateBudget(ctx context.Context, input CreateBudgetInput) (*CostBudget, error) {
	input.ScopeType = strings.TrimSpace(input.ScopeType)
	if input.ScopeType == "" {
		return nil, fmt.Errorf("%w: scope_type is required", ErrValidation)
	}
	if input.Status == "" {
		input.Status = "active"
	}
	input.Currency = defaultCurrency(input.Currency)
	conversion, err := s.convert(ctx, input.Amount, input.Currency, BaseCurrency, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return s.repo.CreateBudget(ctx, input, conversion)
}

func (s *Service) ListBudgets(ctx context.Context, limit int) ([]CostBudget, error) {
	return s.repo.ListBudgets(ctx, limit)
}

func (s *Service) CreateLedgerEntry(ctx context.Context, input CreateLedgerEntryInput) (*CostLedgerEntry, error) {
	normalizeLedgerEntryInput(&input)
	if input.Amount == 0 {
		return nil, fmt.Errorf("%w: amount must not be zero", ErrValidation)
	}
	conversion, err := s.convert(ctx, input.Amount, input.Currency, BaseCurrency, effectiveTime(input.OccurredAt))
	if err != nil {
		return nil, err
	}
	return s.repo.CreateLedgerEntry(ctx, input, conversion)
}

func (s *Service) ListLedgerEntries(ctx context.Context, filter SummaryFilter) ([]CostLedgerEntry, error) {
	return s.repo.ListLedgerEntries(ctx, filter)
}

func (s *Service) Summary(ctx context.Context, filter SummaryFilter) (*CostSummary, error) {
	if filter.Currency == "" {
		filter.Currency = BaseCurrency
	}
	return s.repo.Summary(ctx, filter)
}

func (s *Service) RecordActual(ctx context.Context, input CreateLedgerEntryInput) (*CostLedgerEntry, error) {
	input.LedgerType = LedgerActual
	return s.CreateLedgerEntry(ctx, input)
}

func (s *Service) convert(ctx context.Context, amount float64, fromCurrency, toCurrency string, at time.Time) (ConversionResult, error) {
	fromCurrency = defaultCurrency(fromCurrency)
	toCurrency = defaultCurrency(toCurrency)
	if fromCurrency == toCurrency {
		return ConversionResult{
			Amount:          amount,
			FromCurrency:    fromCurrency,
			ToCurrency:      toCurrency,
			ConvertedAmount: amount,
			Rate:            1,
		}, nil
	}
	rate, err := s.repo.FindExchangeRate(ctx, fromCurrency, toCurrency, at)
	if err == nil {
		return ConversionResult{
			Amount:                amount,
			FromCurrency:          fromCurrency,
			ToCurrency:            toCurrency,
			ConvertedAmount:       amount * rate.Rate,
			Rate:                  rate.Rate,
			ExchangeRateVersionID: &rate.ID,
		}, nil
	}
	if !isNoRows(err) {
		return ConversionResult{}, fmt.Errorf("find exchange rate: %w", err)
	}
	inverse, inverseErr := s.repo.FindExchangeRate(ctx, toCurrency, fromCurrency, at)
	if inverseErr == nil {
		rateValue := 1 / inverse.Rate
		return ConversionResult{
			Amount:                amount,
			FromCurrency:          fromCurrency,
			ToCurrency:            toCurrency,
			ConvertedAmount:       amount * rateValue,
			Rate:                  rateValue,
			ExchangeRateVersionID: &inverse.ID,
		}, nil
	}
	if !isNoRows(inverseErr) {
		return ConversionResult{}, fmt.Errorf("find inverse exchange rate: %w", inverseErr)
	}
	return ConversionResult{}, fmt.Errorf("%w: exchange rate %s to %s is required", ErrValidation, fromCurrency, toCurrency)
}

func normalizeLedgerEntryInput(input *CreateLedgerEntryInput) {
	if input.LedgerType == "" {
		input.LedgerType = LedgerActual
	}
	if input.CostCategory == "" {
		input.CostCategory = "manual"
	}
	if input.SourceType == "" {
		input.SourceType = "manual"
	}
	if input.Status == "" {
		input.Status = "posted"
	}
	input.Currency = defaultCurrency(input.Currency)
	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
}

func defaultCurrency(value string) string {
	value = normalizeCurrency(value)
	if value == "" {
		return BaseCurrency
	}
	return value
}

func normalizeCurrency(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func effectiveTime(value *time.Time) time.Time {
	if value == nil {
		return time.Now().UTC()
	}
	return *value
}
