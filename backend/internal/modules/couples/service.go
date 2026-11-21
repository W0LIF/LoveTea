package couples

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	repo   *Repository
	secret []byte
	redis  *redis.Client
}

var (
	ErrInvalidInvitationInput = errors.New("invalid invitation input")
	ErrInvalidMemberProfile   = errors.New("invalid member profile")
)

func NewService(
	repo *Repository,
	secret string,
	redisClient *redis.Client,
) *Service {
	return &Service{
		repo:   repo,
		secret: []byte(secret),
		redis:  redisClient,
	}
}

func (s *Service) CreateCouple(
	ctx context.Context,
	userID uuid.UUID,
) (CreateCoupleResponse, error) {
	coupleID, err := s.repo.CreateCouple(ctx, userID)
	if err != nil {
		return CreateCoupleResponse{}, err
	}

	return CreateCoupleResponse{
		CoupleID: coupleID.String(),
		Status:   "pending",
	}, nil
}

func (s *Service) CreateInvitation(
	ctx context.Context,
	coupleID uuid.UUID,
	userID uuid.UUID,
) (CreateInvitationResponse, error) {
	codeNumber, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return CreateInvitationResponse{}, err
	}

	code := fmt.Sprintf("%06d", codeNumber.Int64())

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return CreateInvitationResponse{}, err
	}

	token := base64.RawURLEncoding.EncodeToString(tokenBytes)

	codeHash := s.hashCode(code)
	tokenHash := hashToken(token)

	expiresAt := time.Now().UTC().Add(10 * time.Minute)

	invitationID, err := s.repo.CreateInvitation(
		ctx,
		coupleID,
		userID,
		codeHash,
		tokenHash,
		expiresAt,
	)
	if err != nil {
		return CreateInvitationResponse{}, err
	}

	// Открытые code и token возвращаются только один раз.
	return CreateInvitationResponse{
		InvitationID: invitationID.String(),
		Code:         code,
		Token:        token,
		ExpiresAt:    expiresAt,
	}, nil
}

func (s *Service) AcceptInvitation(
	ctx context.Context,
	userID uuid.UUID,
	request AcceptInvitationRequest,
) (AcceptInvitationResponse, error) {
	hasCode := request.Code != ""
	hasToken := request.Token != ""

	if hasCode == hasToken {
		return AcceptInvitationResponse{}, ErrInvalidInvitationInput
	}

	var (
		hash     []byte
		useToken bool
	)

	if hasCode {
		if len(request.Code) != 6 {
			return AcceptInvitationResponse{}, ErrInvalidInvitationInput
		}

		for _, char := range request.Code {
			if char < '0' || char > '9' {
				return AcceptInvitationResponse{}, ErrInvalidInvitationInput
			}
		}

		hash = s.hashCode(request.Code)
	} else {
		if len(request.Token) < 32 || len(request.Token) > 128 {
			return AcceptInvitationResponse{}, ErrInvalidInvitationInput
		}

		hash = hashToken(request.Token)
		useToken = true
	}

	if hasCode {
		if err := s.checkCodeRateLimit(ctx, userID); err != nil {
			return AcceptInvitationResponse{}, err
		}
	}

	coupleID, err := s.repo.AcceptInvitation(
		ctx,
		userID,
		hash,
		useToken,
	)
	if err != nil {
		return AcceptInvitationResponse{}, err
	}

	return AcceptInvitationResponse{
		CoupleID: coupleID.String(),
		Status:   "active",
		Message:  "You have successfully joined the couple",
	}, nil
}

func (s *Service) hashCode(code string) []byte {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(code))
	return mac.Sum(nil)
}

func hashToken(token string) []byte {
	hash := sha256.Sum256([]byte(token))
	return hash[:]
}

func (s *Service) GetMyCouple(
	ctx context.Context,
	userID uuid.UUID,
) (MyCoupleResponse, error) {
	couple, err := s.repo.GetMyCouple(ctx, userID)
	if err != nil {
		return MyCoupleResponse{}, err
	}

	response := MyCoupleResponse{
		CoupleID:    couple.ID.String(),
		Status:      couple.Status,
		CreatedAt:   couple.CreatedAt,
		ActivatedAt: couple.ActivatedAt,
		Members:     make([]CoupleMemberResponse, 0, len(couple.Members)),
	}

	for _, member := range couple.Members {
		response.Members = append(response.Members, CoupleMemberResponse{
			UserID:     member.UserID.String(),
			Email:      member.Email,
			Nickname:   member.Nickname,
			MascotType: member.MascotType,
			JoinedAt:   member.JoinedAt,
		})
	}

	return response, nil
}

func (s *Service) UpdateMemberProfile(
	ctx context.Context,
	userID uuid.UUID,
	request UpdateMemberProfileRequest,
) error {
	if request.Nickname == nil && request.MascotType == nil {
		return ErrInvalidMemberProfile
	}

	if request.Nickname != nil {
		nickname := strings.TrimSpace(*request.Nickname)

		if nickname == "" || utf8.RuneCountInString(nickname) > 50 {
			return ErrInvalidMemberProfile
		}

		request.Nickname = &nickname
	}

	if request.MascotType != nil {
		if *request.MascotType != "kitten" && *request.MascotType != "wolf" {
			return ErrInvalidMemberProfile
		}
	}

	couple, err := s.repo.GetMyCouple(ctx, userID)
	if err != nil {
		return err
	}

	return s.repo.UpdateMemberProfile(
		ctx,
		couple.ID,
		userID,
		request.Nickname,
		request.MascotType,
	)
}

func (s *Service) ConfirmDetach(
	ctx context.Context,
	userID uuid.UUID,
) (DetachResponse, error) {
	coupleID, detached, err := s.repo.ConfirmDetach(ctx, userID)
	if err != nil {
		return DetachResponse{}, err
	}

	if detached {
		return DetachResponse{
			CoupleID:  coupleID.String(),
			Status:    "detached",
			Confirmed: true,
			Message:   "Both partners confirmed. The couple has been detached.",
		}, nil
	}

	return DetachResponse{
		CoupleID:  coupleID.String(),
		Status:    "active",
		Confirmed: false,
		Message:   "Your confirmation has been saved. Waiting for your partner.",
	}, nil
}

func (s *Service) checkCodeRateLimit(
	ctx context.Context,
	userID uuid.UUID,
) error {
	const script = `
        local current = redis.call("INCR", KEYS[1])

        if current == 1 then
            redis.call("EXPIRE", KEYS[1], ARGV[1])
        end

        return current
    `

	key := "love-tea:couple-invite-attempts:" + userID.String()

	result, err := s.redis.Eval(
		ctx,
		script,
		[]string{key},
		int((10 * time.Minute).Seconds()),
	).Int64()

	if err != nil {
		return ErrRateLimitUnavailable
	}

	if result > 5 {
		return ErrInvitationRateLimited
	}

	return nil
}
