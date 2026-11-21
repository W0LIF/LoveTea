package couples

import "time"

type CreateCoupleResponse struct {
	CoupleID string `json:"couple_id"`
	Status   string `json:"status"`
}

type CreateInvitationResponse struct {
	InvitationID string    `json:"invitation_id"`
	Code         string    `json:"code"`
	Token        string    `json:"token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type AcceptInvitationRequest struct {
	Code  string `json:"code,omitempty"`
	Token string `json:"token,omitempty"`
}

type AcceptInvitationResponse struct {
	CoupleID string `json:"couple_id"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

type CoupleMemberResponse struct {
	UserID     string    `json:"user_id"`
	Email      string    `json:"email"`
	Nickname   *string   `json:"nickname"`
	MascotType string    `json:"mascot_type"`
	JoinedAt   time.Time `json:"joined_at"`
}

type MyCoupleResponse struct {
	CoupleID    string                 `json:"couple_id"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	ActivatedAt *time.Time             `json:"activated_at,omitempty"`
	Members     []CoupleMemberResponse `json:"members"`
}

type UpdateMemberProfileRequest struct {
	Nickname   *string `json:"nickname"`
	MascotType *string `json:"mascot_type"`
}

type DetachResponse struct {
	CoupleID  string `json:"couple_id"`
	Status    string `json:"status"`
	Confirmed bool   `json:"confirmed"`
	Message   string `json:"message"`
}
