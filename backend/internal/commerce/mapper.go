package commerce

import "indieforge/internal/dto"

func toPaymentDTO(p Payment, confirmationURL string) dto.PaymentDTO {
	return dto.PaymentDTO{
		ID:              p.ID,
		GameID:          p.GameID,
		UserID:          p.UserID,
		Kind:            p.Kind,
		Amount:          p.Amount,
		Status:          p.Status,
		FriendUsername:  p.FriendUsername,
		ConfirmationURL: confirmationURL,
		CreatedAt:       dto.FormatTime(p.CreatedAt),
	}
}
