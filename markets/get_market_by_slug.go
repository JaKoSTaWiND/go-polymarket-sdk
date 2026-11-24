package markets

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/JaKoSTaWiND/go-polymarket-sdk/internal"
	"github.com/shopspring/decimal"
)

// GetMarketBySlug возвращает рынок Gamma по slug через переданный httpClient.
// Контекст ctx управляет запросом и чтением тела. При ошибке запроса,
// неуспешном HTTP-статусе или ошибке разбора возвращается nil и ошибка.
func GetMarketBySlug(ctx context.Context, httpClient *http.Client, slug string) (*internal.Market, error) {
	endpoint := string(internal.GAMMA_URL) + "/markets/slug/" + url.PathEscape(slug)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("get market %q: HTTP %d", slug, resp.StatusCode)
	}
	return decodeMarket(resp.Body)
}

// gammaStringList хранит элементы массива, который Gamma передаёт внутри JSON-строки.
type gammaStringList []string

// UnmarshalText разбирает содержимое строки text после снятия внешнего JSON-экранирования.
func (list *gammaStringList) UnmarshalText(text []byte) error {
	return json.Unmarshal(text, (*[]string)(list))
}

// decodeMarket читает из body поля рынка и связывает каждый исход с его ценой
// и токеном по общему индексу. EventID берётся из первого события, если оно есть.
func decodeMarket(body io.Reader) (*internal.Market, error) {
	var data *struct {
		ID            string          `json:"id"`
		Question      string          `json:"question"`
		QuestionID    string          `json:"questionID"`
		ConditionID   string          `json:"conditionId"`
		Slug          string          `json:"slug"`
		StartDate     string          `json:"startDate"`
		EndDate       string          `json:"endDate"`
		GameStartTime string          `json:"gameStartTime"`
		Outcomes      gammaStringList `json:"outcomes"`
		Prices        gammaStringList `json:"outcomePrices"`
		TokenIDs      gammaStringList `json:"clobTokenIds"`
		Events        []struct {
			ID string `json:"id"`
		} `json:"events"`
	}
	if err := json.NewDecoder(body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode Gamma market: %w", err)
	}
	if data == nil {
		return nil, fmt.Errorf("expected a Gamma market object, got null")
	}
	if len(data.Outcomes) != len(data.Prices) || len(data.Outcomes) != len(data.TokenIDs) {
		return nil, fmt.Errorf("Gamma market %q: got %d outcomes, %d outcomePrices and %d clobTokenIds",
			data.ID, len(data.Outcomes), len(data.Prices), len(data.TokenIDs))
	}

	market := &internal.Market{
		ID:            data.ID,
		Question:      data.Question,
		QuestionID:    data.QuestionID,
		ConditionId:   data.ConditionID,
		Slug:          data.Slug,
		StartDate:     data.StartDate,
		EndDate:       data.EndDate,
		GameStartTime: data.GameStartTime,
		Tokens:        make([]internal.MarketToken, len(data.Outcomes)),
	}
	if len(data.Events) > 0 {
		market.EventID = data.Events[0].ID
	}
	for i, outcome := range data.Outcomes {
		price, err := decimal.NewFromString(data.Prices[i])
		if err != nil {
			return nil, fmt.Errorf("Gamma market %q outcomePrices[%d]: %w", data.ID, i, err)
		}
		market.Tokens[i] = internal.MarketToken{TokenID: data.TokenIDs[i], Outcome: outcome, Price: price}
	}
	return market, nil
}
