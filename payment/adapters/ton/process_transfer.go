package ton

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	utils "github.com/elum-utils/services/internal/utils"
	"github.com/elum-utils/services/payment/repository"
	paymentsqlc "github.com/elum-utils/services/payment/sqlc"
)

func (a *TON) ProcessTransfer(ctx context.Context, transfer IncomingTransfer) (*ProcessResult, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	transfer.AssetCode = normalizeAsset(transfer.AssetCode)
	network, err := validateNetwork(transfer.Network)
	if err != nil {
		return nil, err
	}
	transfer.Network = network
	transfer.WalletAddress, err = NormalizeWalletAddress(transfer.WalletAddress, network)
	if err != nil {
		return nil, err
	}

	if transfer.TxHash != "" {
		existing, err := a.repository.GetProviderTransactionByExternalID(ctx, paymentsqlc.GetProviderTransactionByExternalIDParams{
			WorkspaceID:           transfer.WorkspaceID,
			ProviderCode:          ProviderCode,
			Network:               transfer.Network,
			SourceKey:             transfer.WalletAddress,
			ExternalTransactionID: transfer.TxHash,
		})
		if err == nil {
			if _, cursorErr := a.repository.UpsertProviderCursor(ctx, paymentsqlc.UpsertProviderCursorParams{
				WorkspaceID:    transfer.WorkspaceID,
				ProviderCode:   ProviderCode,
				Network:        transfer.Network,
				SourceKey:      transfer.WalletAddress,
				CursorValue:    strconv.FormatUint(transfer.LogicalTime, 10),
				CursorSequence: int64(transfer.LogicalTime),
			}); cursorErr != nil {
				return nil, cursorErr
			}
			return &ProcessResult{
				OrderID:     uint64FromNull(existing.OrderID),
				AttemptID:   uint64FromNull(existing.AttemptID),
				Transaction: uint64(existing.ID),
				AlreadyDone: true,
				Ignored:     existing.Status == paymentsqlc.PaymentProviderTransactionStatusIgnored,
			}, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	attempt, err := a.repository.GetAttemptByProviderPaymentID(ctx, ProviderCode, transfer.Comment)
	if err != nil {
		return a.storeTransfer(ctx, transfer, 0, 0, paymentsqlc.PaymentProviderTransactionStatusIgnored, err.Error())
	}
	if attempt.AssetCode != transfer.AssetCode || attempt.AmountMinor != transfer.AmountMinor {
		return a.storeTransfer(ctx, transfer, attempt.OrderID, attempt.ID, paymentsqlc.PaymentProviderTransactionStatusFailed, repository.ErrPaymentMismatch.Error())
	}

	completed, err := a.repository.CompleteAttempt(ctx, repository.CompleteAttemptParams{
		AttemptID:         attempt.ID,
		ProviderCode:      ProviderCode,
		ProviderPaymentID: utils.Ref(transfer.Comment),
		AmountMinor:       transfer.AmountMinor,
		AssetCode:         transfer.AssetCode,
	})
	if err != nil {
		return a.storeTransfer(ctx, transfer, attempt.OrderID, attempt.ID, paymentsqlc.PaymentProviderTransactionStatusFailed, err.Error())
	}

	result, err := a.storeTransfer(ctx, transfer, completed.OrderID, completed.AttemptID, paymentsqlc.PaymentProviderTransactionStatusMatched, "")
	if err != nil {
		return nil, err
	}
	result.AlreadyDone = completed.AlreadyDone
	return result, nil
}

func (a *TON) storeTransfer(ctx context.Context, transfer IncomingTransfer, orderID uint64, attemptID uint64, status paymentsqlc.PaymentProviderTransactionStatus, message string) (*ProcessResult, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	cursor := paymentsqlc.UpsertProviderCursorParams{
		WorkspaceID:    transfer.WorkspaceID,
		ProviderCode:   ProviderCode,
		Network:        transfer.Network,
		SourceKey:      transfer.WalletAddress,
		CursorValue:    strconv.FormatUint(transfer.LogicalTime, 10),
		CursorSequence: int64(transfer.LogicalTime),
	}
	occurredAt := transfer.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}
	id, err := a.repository.StoreProviderTransaction(ctx, paymentsqlc.CreateProviderTransactionParams{
		WorkspaceID:           transfer.WorkspaceID,
		ProviderCode:          ProviderCode,
		Network:               transfer.Network,
		SourceKey:             transfer.WalletAddress,
		AssetCode:             transfer.AssetCode,
		ExternalTransactionID: transfer.TxHash,
		SequenceNumber:        int64(transfer.LogicalTime),
		SourceAddress:         transfer.SourceAddress,
		DestinationAddress:    transfer.DestinationAddress,
		AmountMinor:           int64(transfer.AmountMinor),
		PaymentReference:      transfer.Comment,
		SenderReference:       nullString(transfer.JettonSender),
		OrderID:               nullInt64FromUint64(orderID),
		AttemptID:             nullInt64FromUint64(attemptID),
		Status:                status,
		Error:                 nullString(message),
		OccurredAt:            occurredAt,
	}, cursor)
	if isDuplicateEntry(err) && transfer.TxHash != "" {
		existing, existingErr := a.repository.GetProviderTransactionByExternalID(ctx, paymentsqlc.GetProviderTransactionByExternalIDParams{
			WorkspaceID:           transfer.WorkspaceID,
			ProviderCode:          ProviderCode,
			Network:               transfer.Network,
			SourceKey:             transfer.WalletAddress,
			ExternalTransactionID: transfer.TxHash,
		})
		if existingErr != nil {
			return nil, existingErr
		}
		if _, cursorErr := a.repository.UpsertProviderCursor(ctx, cursor); cursorErr != nil {
			return nil, cursorErr
		}
		return &ProcessResult{
			OrderID:     uint64FromNull(existing.OrderID),
			AttemptID:   uint64FromNull(existing.AttemptID),
			Transaction: uint64(existing.ID),
			AlreadyDone: true,
			Ignored:     existing.Status == paymentsqlc.PaymentProviderTransactionStatusIgnored,
		}, nil
	}
	if err != nil {
		return nil, err
	}

	return &ProcessResult{
		OrderID:     orderID,
		AttemptID:   attemptID,
		Transaction: id,
		Ignored:     status == paymentsqlc.PaymentProviderTransactionStatusIgnored,
	}, nil
}
