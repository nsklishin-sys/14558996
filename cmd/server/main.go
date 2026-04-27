package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net"
	"net/http"
	"net/mail"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"
)

type greetingResponse struct {
	Message string `json:"message"`
}

type user struct {
	ID          int64  `json:"id"`
	PublicID    string `json:"public_id"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	FullName    string `json:"full_name"`
	Email       string `json:"email"`
	Position    string `json:"position,omitempty"`
	CompanyName string `json:"company_name,omitempty"`
	Bio         string `json:"bio,omitempty"`
	Phone       string `json:"phone,omitempty"`
	Location    string `json:"location,omitempty"`
	City        string `json:"city,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	Handle      string `json:"handle,omitempty"`
}

type registerRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Terms     bool   `json:"terms"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type profileUpdateRequest struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	FullName    string `json:"full_name"`
	Position    string `json:"position"`
	CompanyName string `json:"company_name"`
	Bio         string `json:"bio"`
	Phone       string `json:"phone"`
	Location    string `json:"location"`
	City        string `json:"city"`
	AvatarURL   string `json:"avatar_url"`
}

type authResponse struct {
	Token string `json:"token"`
	User  user   `json:"user"`
}

type publicUserProfile struct {
	PublicID     string  `json:"public_id"`
	FirstName    string  `json:"first_name,omitempty"`
	LastName     string  `json:"last_name,omitempty"`
	FullName     string  `json:"full_name"`
	Email        string  `json:"email,omitempty"`
	Handle       string  `json:"handle,omitempty"`
	AvatarURL    string  `json:"avatar_url,omitempty"`
	Position     string  `json:"position,omitempty"`
	CompanyName  string  `json:"company_name,omitempty"`
	Bio          string  `json:"bio,omitempty"`
	Phone        string  `json:"phone,omitempty"`
	Location     string  `json:"location,omitempty"`
	City         string  `json:"city,omitempty"`
	Website      string  `json:"website,omitempty"`
	IsOnline     bool    `json:"is_online"`
	LastSeenAt   *string `json:"last_seen_at,omitempty"`
	IsPrivate    bool    `json:"is_private"`
	CanMessage   bool    `json:"can_message"`
	IsSelf       bool    `json:"is_self"`
	FriendStatus string  `json:"friend_status,omitempty"` // "none" | "outgoing" | "incoming" | "friends" | "self"
}

type userSettings struct {
	NotifEmail            bool   `json:"notif_email"`
	NotifPush             bool   `json:"notif_push"`
	NotifFriendRequests   bool   `json:"notif_friend_requests"`
	NotifChatMessages     bool   `json:"notif_chat_messages"`
	NotifMentions         bool   `json:"notif_mentions"`
	NotifReactions        bool   `json:"notif_reactions"`
	NotifNewJobs          bool   `json:"notif_new_jobs"`
	NotifEvents           bool   `json:"notif_events"`
	NotifNewsDigest       bool   `json:"notif_news_digest"`
	NotifPlatformUpdates  bool   `json:"notif_platform_updates"`
	PrivacyProfilePrivate bool   `json:"privacy_profile_private"`
	PrivacyShowEmail      bool   `json:"privacy_show_email"`
	PrivacyShowPhone      bool   `json:"privacy_show_phone"`
	PrivacyDiscoverable   bool   `json:"privacy_discoverable"`
	PrivacyShowOnline     bool   `json:"privacy_show_online"`
	PrivacyShowLastSeen   bool   `json:"privacy_show_last_seen"`
	PrivacyWhoCanMessage  string `json:"privacy_who_can_message"`
	Theme                 string `json:"theme"`
	LayoutMode            string `json:"layout_mode"`
	CompactFeed           bool   `json:"compact_feed"`
	Locale                string `json:"locale"`
	Timezone              string `json:"timezone"`
	DateFormat            string `json:"date_format"`
}

type updateHandleRequest struct {
	Handle string `json:"handle"`
}

type changeEmailRequest struct {
	Email           string `json:"email"`
	CurrentPassword string `json:"current_password"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type deleteAccountRequest struct {
	CurrentPassword string `json:"current_password"`
	Confirmation    string `json:"confirmation"`
}

type updateNotificationsRequest struct {
	NotifEmail           *bool `json:"notif_email,omitempty"`
	NotifPush            *bool `json:"notif_push,omitempty"`
	NotifFriendRequests  *bool `json:"notif_friend_requests,omitempty"`
	NotifChatMessages    *bool `json:"notif_chat_messages,omitempty"`
	NotifMentions        *bool `json:"notif_mentions,omitempty"`
	NotifReactions       *bool `json:"notif_reactions,omitempty"`
	NotifNewJobs         *bool `json:"notif_new_jobs,omitempty"`
	NotifEvents          *bool `json:"notif_events,omitempty"`
	NotifNewsDigest      *bool `json:"notif_news_digest,omitempty"`
	NotifPlatformUpdates *bool `json:"notif_platform_updates,omitempty"`
}

type updatePrivacyRequest struct {
	PrivacyProfilePrivate *bool   `json:"privacy_profile_private,omitempty"`
	PrivacyShowEmail      *bool   `json:"privacy_show_email,omitempty"`
	PrivacyShowPhone      *bool   `json:"privacy_show_phone,omitempty"`
	PrivacyDiscoverable   *bool   `json:"privacy_discoverable,omitempty"`
	PrivacyShowOnline     *bool   `json:"privacy_show_online,omitempty"`
	PrivacyShowLastSeen   *bool   `json:"privacy_show_last_seen,omitempty"`
	PrivacyWhoCanMessage  *string `json:"privacy_who_can_message,omitempty"`
}

type updateAppearanceRequest struct {
	Theme       *string `json:"theme,omitempty"`
	LayoutMode  *string `json:"layout_mode,omitempty"`
	CompactFeed *bool   `json:"compact_feed,omitempty"`
	Locale      *string `json:"locale,omitempty"`
	Timezone    *string `json:"timezone,omitempty"`
	DateFormat  *string `json:"date_format,omitempty"`
}

type friendDTO struct {
	FriendID    string `json:"friend_id"`
	FriendName  string `json:"friend_name"`
	Email       string `json:"email"`
	Position    string `json:"position,omitempty"`
	CompanyName string `json:"company_name,omitempty"`
}

type friendCandidateDTO struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}

type community struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Region      string    `json:"region"`
	Website     string    `json:"website"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	AvatarURL   string    `json:"avatar_url"`
	CoverURL    string    `json:"cover_url"`
	Color       string    `json:"color"`
	Tags        []string  `json:"tags"`
	Privacy     string    `json:"privacy_level"`
	IsMember    bool      `json:"is_member"`
	MyRole      string    `json:"my_role,omitempty"`
	Members     int       `json:"members_count"`
	CreatorID   string    `json:"creator_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type createCommunityRequest struct {
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	Description  string   `json:"description"`
	Region       string   `json:"region"`
	Website      string   `json:"website"`
	Email        string   `json:"email"`
	Phone        string   `json:"phone"`
	Tags         []string `json:"tags"`
	PrivacyLevel string   `json:"privacy_level"`
	AvatarURL    string   `json:"avatar_url"`
	Color        string   `json:"color"`
}

type updateCommunityRequest struct {
	Name         *string   `json:"name,omitempty"`
	Category     *string   `json:"category,omitempty"`
	Description  *string   `json:"description,omitempty"`
	Region       *string   `json:"region,omitempty"`
	Website      *string   `json:"website,omitempty"`
	Email        *string   `json:"email,omitempty"`
	Phone        *string   `json:"phone,omitempty"`
	Tags         *[]string `json:"tags,omitempty"`
	PrivacyLevel *string   `json:"privacy_level,omitempty"`
	AvatarURL    *string   `json:"avatar_url,omitempty"`
	CoverURL     *string   `json:"cover_url,omitempty"`
	Color        *string   `json:"color,omitempty"`
}

type communityMemberDTO struct {
	UserPublicID string    `json:"user_public_id"`
	FullName     string    `json:"full_name"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	Role         string    `json:"role"`
	JoinedAt     time.Time `json:"joined_at"`
}

type communityJoinRequestDTO struct {
	ID           int64     `json:"id"`
	UserPublicID string    `json:"user_public_id"`
	FullName     string    `json:"full_name"`
	Message      string    `json:"message"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type changeRoleRequest struct {
	Role string `json:"role"`
}

type joinRequestBody struct {
	Message string `json:"message"`
}

type searchUser struct {
	PublicID    string  `json:"public_id"`
	FullName    string  `json:"full_name"`
	Handle      string  `json:"handle"`
	Position    string  `json:"position"`
	CompanyName string  `json:"company_name"`
	Bio         string  `json:"bio"`
	City        string  `json:"city"`
	Score       float64 `json:"score,omitempty"`
}

type searchPost struct {
	PublicID       string    `json:"public_id"`
	Content        string    `json:"content"`
	Category       string    `json:"category"`
	AuthorPublicID string    `json:"author_public_id"`
	AuthorName     string    `json:"author_name"`
	LikesCount     int       `json:"likes_count"`
	CommentsCount  int       `json:"comments_count"`
	CreatedAt      time.Time `json:"created_at"`
	Score          float64   `json:"score,omitempty"`
}

type searchCommunity struct {
	PublicID     string  `json:"public_id"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Region       string  `json:"region"`
	Category     string  `json:"category"`
	Color        string  `json:"color"`
	MembersCount int     `json:"members_count"`
	Privacy      string  `json:"privacy"`
	Score        float64 `json:"score,omitempty"`
}

type searchEvent struct {
	PublicID        string    `json:"public_id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Type            string    `json:"type"`
	Format          string    `json:"format"`
	City            string    `json:"city"`
	StartsAt        time.Time `json:"starts_at"`
	RegisteredCount int       `json:"registered_count"`
	BannerColor     string    `json:"banner_color"`
	Score           float64   `json:"score,omitempty"`
}

type searchCompany struct {
	Name        string  `json:"name"`
	Industry    string  `json:"industry"`
	City        string  `json:"city"`
	IsPartner   bool    `json:"is_partner"`
	Description string  `json:"description"`
	Score       float64 `json:"score,omitempty"`
}

type searchSection struct{}

type searchResult struct {
	Query       string            `json:"query"`
	Total       int               `json:"total"`
	TookMS      int64             `json:"took_ms"`
	Counts      map[string]int    `json:"counts"`
	Users       []searchUser      `json:"users,omitempty"`
	Posts       []searchPost      `json:"posts,omitempty"`
	Communities []searchCommunity `json:"communities,omitempty"`
	Events      []searchEvent     `json:"events,omitempty"`
	Companies   []searchCompany   `json:"companies,omitempty"`
	Sections    []searchSection   `json:"sections"`
}

type chatConversation struct {
	ID                int64      `json:"id"`
	PublicID          string     `json:"public_id"`
	Type              string     `json:"type"`
	Title             string     `json:"title"`
	AvatarURL         string     `json:"avatar_url"`
	CommunityID       *int64     `json:"community_id,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	LastMessageAt     *time.Time `json:"last_message_at,omitempty"`
	DisplayName       string     `json:"display_name"`
	DisplayRole       string     `json:"display_role"`
	DisplayColor      string     `json:"display_color"`
	DisplayAvatar     string     `json:"display_avatar"`
	OtherPublicID     string     `json:"other_public_id,omitempty"`
	LastMessageText   string     `json:"last_message_text"`
	LastMessageAuthor string     `json:"last_message_author"`
	UnreadCount       int        `json:"unread_count"`
	MembersCount      int        `json:"members_count"`
	IsOnline          bool       `json:"is_online"`
	Pinned            bool       `json:"pinned"`
	Muted             bool       `json:"muted"`
	MyRole            string     `json:"my_role"`
}

type chatReplyPreview struct {
	ID             int64  `json:"id"`
	AuthorName     string `json:"author_name"`
	ContentPreview string `json:"content_preview"`
}

type chatMessage struct {
	ID             int64             `json:"id"`
	ConversationID int64             `json:"conversation_id"`
	AuthorPublicID string            `json:"author_public_id"`
	AuthorName     string            `json:"author_name"`
	AuthorColor    string            `json:"author_color"`
	AuthorAvatar   string            `json:"author_avatar,omitempty"`
	Content        string            `json:"content"`
	AttachmentURL  string            `json:"attachment_url,omitempty"`
	AttachmentName string            `json:"attachment_name,omitempty"`
	AttachmentType string            `json:"attachment_type,omitempty"`
	AttachmentSize int64             `json:"attachment_size,omitempty"`
	ReplyTo        *chatReplyPreview `json:"reply_to,omitempty"`
	IsEdited       bool              `json:"is_edited"`
	IsDeleted      bool              `json:"is_deleted"`
	CreatedAt      time.Time         `json:"created_at"`
	EditedAt       *time.Time        `json:"edited_at,omitempty"`
	IsMine         bool              `json:"is_mine"`
}

type createDirectConversationRequest struct {
	UserPublicID string `json:"user_public_id"`
}
type createGroupConversationRequest struct {
	Title     string   `json:"title"`
	MemberIDs []string `json:"member_public_ids"`
}
type sendMessageRequest struct {
	Content        string `json:"content"`
	ReplyToID      *int64 `json:"reply_to_id,omitempty"`
	AttachmentURL  string `json:"attachment_url,omitempty"`
	AttachmentName string `json:"attachment_name,omitempty"`
	AttachmentType string `json:"attachment_type,omitempty"`
	AttachmentSize int64  `json:"attachment_size,omitempty"`
}
type editMessageRequest struct {
	Content string `json:"content"`
}
type typingRequest struct {
	IsTyping bool `json:"is_typing"`
}
type markReadRequest struct {
	MessageID int64 `json:"message_id"`
}
type toggleBoolRequest struct {
	Pinned bool `json:"pinned"`
	Muted  bool `json:"muted"`
}

type chatPresenceStore struct {
	mu   sync.Mutex
	last map[int64]time.Time
}

func newChatPresenceStore() *chatPresenceStore {
	return &chatPresenceStore{last: map[int64]time.Time{}}
}
func (s *chatPresenceStore) touch(uid int64) {
	s.mu.Lock()
	s.last[uid] = time.Now().UTC()
	s.mu.Unlock()
}
func (s *chatPresenceStore) isOnline(uid int64) bool {
	s.mu.Lock()
	t := s.last[uid]
	s.mu.Unlock()
	return time.Since(t) < 30*time.Second
}

func newChatPublicID() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "c" + hex.EncodeToString(b), nil
}
func stableColorForName(name string) string {
	pal := []string{"#5AB080", "#3A90C0", "#9060C0", "#C07030", "#1A8A6A", "#B05090", "#208090", "#3B6D11"}
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.ToLower(strings.TrimSpace(name))))
	return pal[int(h.Sum32())%len(pal)]
}
func messagePreview(content string, maxRunes int) string {
	c := strings.ReplaceAll(strings.TrimSpace(content), "\n", " ")
	if utf8.RuneCountInString(c) <= maxRunes {
		return c
	}
	rs := []rune(c)
	return string(rs[:maxRunes]) + "…"
}

func chatEscapeILike(q string) string {
	q = strings.ReplaceAll(q, "\\", "\\\\")
	q = strings.ReplaceAll(q, "%", "\\%")
	q = strings.ReplaceAll(q, "_", "\\_")
	return q
}

type post struct {
	ID             int64     `json:"id"`
	PublicID       string    `json:"public_id"`
	Type           string    `json:"type"`
	Title          string    `json:"title"`
	Content        string    `json:"content"`
	Text           string    `json:"text"`
	CoverURL       string    `json:"cover_url,omitempty"`
	Tags           []string  `json:"tags"`
	PrivacyLevel   string    `json:"privacy_level"`
	LikesCount     int       `json:"likes_count"`
	CommentsCount  int       `json:"comments_count"`
	ViewsCount     int       `json:"views_count"`
	SavesCount     int       `json:"saves_count"`
	RepostsCount   int       `json:"reposts_count"`
	IsLiked        bool      `json:"is_liked"`
	IsSaved        bool      `json:"is_saved"`
	IsReposted     bool      `json:"is_reposted"`
	RepostedFromID int64     `json:"reposted_from_id,omitempty"`
	Repost         *post     `json:"repost,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	AuthorID       int64     `json:"-"`
	AuthorPublicID string    `json:"author_public_id"`
	AuthorName     string    `json:"author"`
	AuthorRole     string    `json:"author_role"`
	AuthorAvatar   string    `json:"author_avatar,omitempty"`
	CommunityID    int64     `json:"community_id,omitempty"`
	CommunityName  string    `json:"community_name,omitempty"`
}

type notification struct {
	ID             int64     `json:"id"`
	Type           string    `json:"type"`
	SourceType     string    `json:"source_type,omitempty"`
	SourceID       int64     `json:"source_id,omitempty"`
	SourcePublicID string    `json:"source_public_id,omitempty"`
	Title          string    `json:"title"`
	Preview        string    `json:"preview,omitempty"`
	IsRead         bool      `json:"is_read"`
	CreatedAt      time.Time `json:"created_at"`

	ActorPublicID string `json:"actor_public_id,omitempty"`
	ActorName     string `json:"actor_name,omitempty"`
	ActorColor    string `json:"actor_color,omitempty"`
	ActorAvatar   string `json:"actor_avatar,omitempty"`
}

type createNotificationParams struct {
	RecipientID    int64
	ActorID        int64
	Type           string
	SourceType     string
	SourceID       int64
	SourcePublicID string
	Title          string
	Preview        string
}

type createPostRequest struct {
	Title        string   `json:"title"`
	Content      string   `json:"content"`
	Type         string   `json:"type"`
	Tags         []string `json:"tags"`
	CoverURL     string   `json:"cover_url"`
	PrivacyLevel string   `json:"privacy_level"`
}

type project struct {
	ID            int64      `json:"id"`
	PublicID      string     `json:"public_id"`
	OwnerID       int64      `json:"-"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Category      string     `json:"category,omitempty"`
	Status        string     `json:"status"`
	Deadline      *time.Time `json:"deadline,omitempty"`
	Budget        *int64     `json:"budget,omitempty"`
	CoverColor    string     `json:"cover_color"`
	Tags          []string   `json:"tags"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	OwnerPublicID string     `json:"owner_public_id,omitempty"`
	OwnerName     string     `json:"owner_name,omitempty"`
	OwnerAvatar   string     `json:"owner_avatar,omitempty"`
	MembersCount  int        `json:"members_count"`
	IsMember      bool       `json:"is_member"`
	IsOwner       bool       `json:"is_owner"`
}

type projectMember struct {
	UserPublicID string    `json:"user_public_id"`
	UserName     string    `json:"user_name"`
	UserAvatar   string    `json:"user_avatar,omitempty"`
	UserColor    string    `json:"user_color"`
	Role         string    `json:"role"`
	JoinedAt     time.Time `json:"joined_at"`
}

type createProjectRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Category    string     `json:"category"`
	Status      string     `json:"status"`
	Deadline    *time.Time `json:"deadline,omitempty"`
	Budget      *int64     `json:"budget,omitempty"`
	CoverColor  string     `json:"cover_color"`
	Tags        []string   `json:"tags"`
}

type updateProjectRequest struct {
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	Category    *string    `json:"category,omitempty"`
	Status      *string    `json:"status,omitempty"`
	Deadline    *time.Time `json:"deadline,omitempty"`
	Budget      *int64     `json:"budget,omitempty"`
	CoverColor  *string    `json:"cover_color,omitempty"`
	Tags        *[]string  `json:"tags,omitempty"`
}

type addProjectMemberRequest struct {
	UserPublicID string `json:"user_public_id"`
	Role         string `json:"role,omitempty"`
}

type event struct {
	ID              int64     `json:"id"`
	PublicID        string    `json:"public_id"`
	OrganizerID     int64     `json:"-"`
	OrganizerPublic string    `json:"organizer_public_id"`
	OrganizerName   string    `json:"organizer_name"`
	CommunityID     *int64    `json:"community_id,omitempty"`
	CommunityName   string    `json:"community_name,omitempty"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Type            string    `json:"type"`
	Format          string    `json:"format"`
	Category        string    `json:"category"`
	City            string    `json:"city"`
	Address         string    `json:"address"`
	Venue           string    `json:"venue"`
	OnlineURL       string    `json:"online_url,omitempty"`
	StartsAt        time.Time `json:"starts_at"`
	EndsAt          time.Time `json:"ends_at"`
	Timezone        string    `json:"timezone"`
	CoverURL        string    `json:"cover_url,omitempty"`
	BannerColor     string    `json:"banner_color,omitempty"`
	FeeCents        int       `json:"fee_cents"`
	Currency        string    `json:"currency"`
	SeatsTotal      int       `json:"seats_total"`
	RegisteredCount int       `json:"registered_count"`
	SavedCount      int       `json:"saved_count"`
	ViewsCount      int       `json:"views_count"`
	Tags            []string  `json:"tags"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	IsRegistered    bool      `json:"is_registered"`
	IsSaved         bool      `json:"is_saved"`
	IsMine          bool      `json:"is_mine"`
}

type createEventRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Format      string   `json:"format"`
	Category    string   `json:"category"`
	City        string   `json:"city"`
	Address     string   `json:"address"`
	Venue       string   `json:"venue"`
	OnlineURL   string   `json:"online_url"`
	StartsAt    string   `json:"starts_at"`
	EndsAt      string   `json:"ends_at"`
	Timezone    string   `json:"timezone"`
	CoverURL    string   `json:"cover_url"`
	BannerColor string   `json:"banner_color"`
	FeeCents    int      `json:"fee_cents"`
	Currency    string   `json:"currency"`
	SeatsTotal  int      `json:"seats_total"`
	Tags        []string `json:"tags"`
	CommunityID *int64   `json:"community_id,omitempty"`
}

type updateEventRequest struct {
	Title       *string   `json:"title,omitempty"`
	Description *string   `json:"description,omitempty"`
	Type        *string   `json:"type,omitempty"`
	Format      *string   `json:"format,omitempty"`
	Category    *string   `json:"category,omitempty"`
	City        *string   `json:"city,omitempty"`
	Address     *string   `json:"address,omitempty"`
	Venue       *string   `json:"venue,omitempty"`
	OnlineURL   *string   `json:"online_url,omitempty"`
	StartsAt    *string   `json:"starts_at,omitempty"`
	EndsAt      *string   `json:"ends_at,omitempty"`
	Timezone    *string   `json:"timezone,omitempty"`
	CoverURL    *string   `json:"cover_url,omitempty"`
	BannerColor *string   `json:"banner_color,omitempty"`
	FeeCents    *int      `json:"fee_cents,omitempty"`
	Currency    *string   `json:"currency,omitempty"`
	SeatsTotal  *int      `json:"seats_total,omitempty"`
	Tags        *[]string `json:"tags,omitempty"`
	Status      *string   `json:"status,omitempty"`
}

type registerToEventRequest struct {
	TicketType string `json:"ticket_type"`
}

type postComment struct {
	ID             int64     `json:"id"`
	AuthorPublicID string    `json:"author_public_id"`
	AuthorName     string    `json:"author_name"`
	Content        string    `json:"content"`
	ParentID       *int64    `json:"parent_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type createCommentRequest struct {
	Content  string `json:"content"`
	ParentID *int64 `json:"parent_id,omitempty"`
}

type sessionStore struct {
	db    *sql.DB
	cache *sessionCache
}

type sessionCache struct {
	mu    sync.RWMutex
	items map[string]sessionCacheEntry
}

type sessionCacheEntry struct {
	userID   int64
	cachedAt time.Time
}

// statsCache — простой in-memory кеш для /api/stats.
// TTL 60 секунд. Не нужен для production-уровня, но избавляет от частых COUNT'ов.
type statsCache struct {
	mu      sync.RWMutex
	payload []byte
	fetched time.Time
}

const sessionCacheTTL = 10 * time.Minute
const statsCacheTTL = 60 * time.Second
const sessionLifetime = 30 * 24 * time.Hour

var globalStatsCache = &statsCache{}

type ipRateLimiter struct {
	mu      sync.Mutex
	limit   rate.Limit
	burst   int
	entries map[string]*rateEntry
}

type rateEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

// htmlInject инжектируется во все HTML-ответы перед </head>.
// Содержит:
//  1. CSS для плавного фейда между страницами + View Transitions API.
//  2. JS для перехвата href="/" (замена на /home-auth.html для залогиненного,
//     на /home-guest.html для гостя) — это убирает белую страницу index.html
//     на клике "Главная".
//  3. Редирект залогиненного юзера с /home-guest.html и /index.html
//     сразу на /home-auth.html, без рендера гостевой.
const htmlInject = `<style>
@view-transition { navigation: auto; }
::view-transition-old(root),
::view-transition-new(root) {
  animation-duration: 180ms;
  animation-timing-function: cubic-bezier(0.25, 0.8, 0.25, 1);
}
html.pt-leaving body {
  opacity: 0;
  transition: opacity 140ms cubic-bezier(0.4, 0, 0.2, 1);
}
@keyframes pt-fade-in { from{opacity:0} to{opacity:1} }
html.pt-ready body {
  animation: pt-fade-in 160ms cubic-bezier(0.4, 0, 0.2, 1) both;
}
@media (prefers-reduced-motion: reduce) {
  ::view-transition-old(root),
  ::view-transition-new(root) { animation-duration: 0.01ms !important; }
  html.pt-leaving body, html.pt-ready body {
    animation: none !important;
    transition: none !important;
    opacity: 1 !important;
  }
}
</style>
<script>
(function(){
  'use strict';
  var html = document.documentElement;

  function hasToken(){
    try { var t = localStorage.getItem('token'); return !!(t && t.trim()); }
    catch(_) { return false; }
  }
  function prm(){
    return window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  }
  function hasVT(){
    return typeof document.startViewTransition === 'function' ||
      (window.CSS && CSS.supports && CSS.supports('view-transition-name: a'));
  }

  // Мгновенный редирект залогиненного юзера с гостевых страниц
  var path = location.pathname;
  if ((path === '/home-guest.html' || path === '/index.html' || path === '/') && hasToken()) {
    location.replace('/home-auth.html');
    return;
  }

  if (!prm()) html.classList.add('pt-ready');

  function isInternalNav(a){
    if (!a || a.tagName !== 'A') return false;
    if (a.target && a.target !== '' && a.target !== '_self') return false;
    if (a.hasAttribute('download')) return false;
    var raw = a.getAttribute('href');
    if (!raw) return false;
    if (raw.charAt(0) === '#') return false;
    if (/^(mailto:|tel:|javascript:)/i.test(raw)) return false;
    try {
      var u = new URL(a.href, location.href);
      if (u.origin !== location.origin) return false;
      if (u.pathname === location.pathname && u.search === location.search && u.hash) return false;
      return true;
    } catch(_) { return false; }
  }

  // "/" → /home-auth.html для залогиненного, /home-guest.html для гостя
  // /index.html и /home-guest.html для залогиненного → /home-auth.html
  function resolveHome(u){
    var p = u.pathname;
    var isHome = p === '/' || p === '/index.html' || p === '/home-guest.html';
    if (!isHome) return null;
    var target = hasToken() ? '/home-auth.html' : '/home-guest.html';
    if (p === target) return null;
    return target + u.search + u.hash;
  }

  var useFade = !prm() && !hasVT();
  var FADE = 140;

  document.addEventListener('click', function(e){
    if (e.defaultPrevented) return;
    if (e.button !== 0) return;
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
    var a = e.target && e.target.closest ? e.target.closest('a') : null;
    if (!isInternalNav(a)) return;

    var u;
    try { u = new URL(a.href, location.href); } catch(_) { return; }

    var home = resolveHome(u);
    var finalHref = home != null ? new URL(home, location.href).href : a.href;

    if (!useFade) {
      if (home != null) { e.preventDefault(); location.href = finalHref; }
      return;
    }

    e.preventDefault();
    html.classList.add('pt-leaving');
    var fb = setTimeout(function(){ html.classList.remove('pt-leaving'); }, 500);
    setTimeout(function(){ clearTimeout(fb); location.href = finalHref; }, FADE);
  }, true);

  window.addEventListener('pageshow', function(){ html.classList.remove('pt-leaving'); });
})();
</script>
<link rel="stylesheet" href="/assets/dark-theme.css">
<script src="/assets/dark-theme-init.js"></script>
<script src="/assets/global-settings.js" defer></script>
<script src="/site-search.js" defer></script>`

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func newIPRateLimiter(limit int, interval time.Duration) *ipRateLimiter {
	return &ipRateLimiter{
		limit:   rate.Every(interval / time.Duration(limit)),
		burst:   limit,
		entries: make(map[string]*rateEntry),
	}
}

func (l *ipRateLimiter) Allow(ip string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.entries[ip]
	if !ok {
		entry = &rateEntry{
			limiter: rate.NewLimiter(l.limit, l.burst),
		}
		l.entries[ip] = entry
	}
	entry.lastSeen = now

	if len(l.entries) > 10000 {
		next := make(map[string]*rateEntry, len(l.entries))
		for key, item := range l.entries {
			if now.Sub(item.lastSeen) <= 10*time.Minute {
				next[key] = item
			}
		}
		l.entries = next
	}

	return entry.limiter.Allow()
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func newSessionStore(db *sql.DB) *sessionStore {
	s := &sessionStore{
		db: db,
		cache: &sessionCache{
			items: make(map[string]sessionCacheEntry),
		},
	}
	go s.cleanupLoop()
	return s
}

func (s *sessionStore) put(token string, userID int64) {
	th := hashToken(token)
	expires := time.Now().Add(sessionLifetime)

	_, err := s.db.Exec(`
		INSERT INTO sessions (token_hash, user_id, expires_at, last_seen_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (token_hash) DO UPDATE
		SET last_seen_at = NOW(), expires_at = EXCLUDED.expires_at
	`, th, userID, expires)
	if err != nil {
		log.Printf("[sessions] put failed: %v", err)
	}

	s.cache.mu.Lock()
	s.cache.items[th] = sessionCacheEntry{userID: userID, cachedAt: time.Now()}
	s.cache.mu.Unlock()
}

func (s *sessionStore) getUserID(token string) (int64, bool) {
	if token == "" {
		return 0, false
	}

	th := hashToken(token)

	s.cache.mu.RLock()
	if entry, ok := s.cache.items[th]; ok {
		if time.Since(entry.cachedAt) < sessionCacheTTL {
			s.cache.mu.RUnlock()
			return entry.userID, true
		}
	}
	s.cache.mu.RUnlock()

	var userID int64
	var expiresAt time.Time
	err := s.db.QueryRow(`
		SELECT user_id, expires_at FROM sessions WHERE token_hash = $1
	`, th).Scan(&userID, &expiresAt)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("[sessions] getUserID failed: %v", err)
		}
		return 0, false
	}

	if time.Now().After(expiresAt) {
		_, _ = s.db.Exec(`DELETE FROM sessions WHERE token_hash = $1`, th)
		s.cache.mu.Lock()
		delete(s.cache.items, th)
		s.cache.mu.Unlock()
		return 0, false
	}

	go func() {
		_, _ = s.db.Exec(`UPDATE sessions SET last_seen_at = NOW() WHERE token_hash = $1`, th)
	}()

	s.cache.mu.Lock()
	s.cache.items[th] = sessionCacheEntry{userID: userID, cachedAt: time.Now()}
	s.cache.mu.Unlock()

	return userID, true
}

func (s *sessionStore) invalidateUser(userID int64) {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE user_id = $1`, userID)
	if err != nil {
		log.Printf("[sessions] invalidateUser failed: %v", err)
	}

	s.cache.mu.Lock()
	for th, entry := range s.cache.items {
		if entry.userID == userID {
			delete(s.cache.items, th)
		}
	}
	s.cache.mu.Unlock()
}

func (s *sessionStore) invalidateUserExcept(userID int64, exceptToken string) {
	exceptHash := hashToken(exceptToken)
	_, err := s.db.Exec(`
		DELETE FROM sessions WHERE user_id = $1 AND token_hash != $2
	`, userID, exceptHash)
	if err != nil {
		log.Printf("[sessions] invalidateUserExcept failed: %v", err)
	}

	s.cache.mu.Lock()
	for th, entry := range s.cache.items {
		if entry.userID == userID && th != exceptHash {
			delete(s.cache.items, th)
		}
	}
	s.cache.mu.Unlock()
}

func (s *sessionStore) cleanupLoop() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		s.cleanupExpired()
	}
}

func (s *sessionStore) cleanupExpired() {
	res, err := s.db.Exec(`DELETE FROM sessions WHERE expires_at < NOW()`)
	if err != nil {
		log.Printf("[sessions] cleanup failed: %v", err)
		return
	}

	n, _ := res.RowsAffected()
	if n > 0 {
		log.Printf("[sessions] cleaned up %d expired sessions", n)
	}

	cutoff := time.Now().Add(-sessionCacheTTL)
	s.cache.mu.Lock()
	for th, entry := range s.cache.items {
		if entry.cachedAt.Before(cutoff) {
			delete(s.cache.items, th)
		}
	}
	s.cache.mu.Unlock()
}

func main() {
	db, err := initDBFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	sessions := newSessionStore(db)
	authRateLimiter := newIPRateLimiter(10, time.Minute)
	postRateLimiter := newIPRateLimiter(10, time.Minute)
	commentRateLimiter := newIPRateLimiter(30, time.Minute)
	communityCreateRateLimiter := newIPRateLimiter(5, time.Hour)
	communityPostRateLimiter := newIPRateLimiter(20, time.Hour)
	communityJoinRequestRateLimiter := newIPRateLimiter(10, time.Hour)
	handleCheckLimiter := newIPRateLimiter(30, time.Minute)
	handleUpdateLimiter := newIPRateLimiter(5, 24*time.Hour)
	accountChangeLimiter := newIPRateLimiter(5, time.Hour)
	accountDeleteLimiter := newIPRateLimiter(3, 24*time.Hour)
	eventCreateLimiter := newIPRateLimiter(5, time.Hour)
	eventRegisterLimiter := newIPRateLimiter(30, time.Minute)
	eventSaveLimiter := newIPRateLimiter(60, time.Minute)
	eventPatchLimiter := newIPRateLimiter(30, time.Hour)
	eventDeleteLimiter := newIPRateLimiter(10, 24*time.Hour)
	searchLimiter := newIPRateLimiter(60, time.Minute)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			writeError(w, http.StatusServiceUnavailable, "database down")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "database": "up"})
	})

	mux.HandleFunc("/api/greeting", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, greetingResponse{Message: "Привет! Добро пожаловать в новый проект на Go + TypeScript 🚀"})
	})

	mux.HandleFunc("/api/auth/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		if !authRateLimiter.Allow(clientIP(r)) {
			writeError(w, http.StatusTooManyRequests, "Слишком много попыток, попробуйте позже")
			return
		}

		var req registerRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "Некорректный JSON")
			return
		}

		createdUser, err := createUser(db, req)
		if err != nil {
			if errors.Is(err, errValidation) {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			if errors.Is(err, errEmailTaken) {
				writeError(w, http.StatusConflict, "Пользователь с таким email уже существует")
				return
			}
			log.Printf("register error: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}

		token, err := newToken()
		if err != nil {
			log.Printf("token generate error: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}

		sessions.put(token, createdUser.ID)
		writeJSON(w, http.StatusCreated, authResponse{Token: token, User: createdUser})
	})

	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		if !authRateLimiter.Allow(clientIP(r)) {
			writeError(w, http.StatusTooManyRequests, "Слишком много попыток, попробуйте позже")
			return
		}

		var req loginRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "Некорректный JSON")
			return
		}

		authUser, err := loginUser(db, req)
		if err != nil {
			if errors.Is(err, errInvalidCredentials) {
				writeError(w, http.StatusUnauthorized, "Неверный email или пароль")
				return
			}
			if errors.Is(err, errValidation) {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			log.Printf("login error: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}

		token, err := newToken()
		if err != nil {
			log.Printf("token generate error: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}

		sessions.put(token, authUser.ID)
		writeJSON(w, http.StatusOK, authResponse{Token: token, User: authUser})
	})

	mux.HandleFunc("/api/me", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}

		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}

		authUser, err := getUserByID(db, userID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusUnauthorized, "Пользователь не найден")
				return
			}
			log.Printf("get current user error: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"user": authUser})
	})

	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}

		globalStatsCache.mu.RLock()
		if globalStatsCache.payload != nil && time.Since(globalStatsCache.fetched) < statsCacheTTL {
			cached := append([]byte(nil), globalStatsCache.payload...)
			globalStatsCache.mu.RUnlock()
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "public, max-age=60")
			_, _ = w.Write(cached)
			return
		}
		globalStatsCache.mu.RUnlock()

		stats, err := computePlatformStats(db)
		if err != nil {
			log.Printf("computePlatformStats failed: %v", err)
			writeError(w, http.StatusInternalServerError, "Не удалось получить статистику")
			return
		}

		payload, err := json.Marshal(stats)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Ошибка сериализации")
			return
		}

		globalStatsCache.mu.Lock()
		globalStatsCache.payload = append([]byte(nil), payload...)
		globalStatsCache.fetched = time.Now()
		globalStatsCache.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=60")
		_, _ = w.Write(payload)
	})

	mux.HandleFunc("/api/me/saved", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, hasAuth := authenticatedUserID(w, r, sessions)
		if !hasAuth {
			return
		}
		limit := parseLimit(r.URL.Query().Get("limit"), 30, 100)
		posts, err := listSavedPosts(db, userID, limit)
		if err != nil {
			handlePostActionError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"posts": posts})
	})

	mux.HandleFunc("/api/profile", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}

		switch r.Method {
		case http.MethodGet:
			u, err := getUserByID(db, userID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					writeError(w, http.StatusNotFound, "Пользователь не найден")
					return
				}
				log.Printf("get profile error: %v", err)
				writeError(w, http.StatusInternalServerError, "Ошибка сервера")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"user": u})
		case http.MethodPut:
			var req profileUpdateRequest
			if err := decodeJSON(w, r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "Некорректный JSON")
				return
			}
			updatedUser, err := updateUserProfile(db, userID, req)
			if err != nil {
				if errors.Is(err, errValidation) {
					writeError(w, http.StatusBadRequest, err.Error())
					return
				}
				if errors.Is(err, sql.ErrNoRows) {
					writeError(w, http.StatusNotFound, "Пользователь не найден")
					return
				}
				log.Printf("update profile error: %v", err)
				writeError(w, http.StatusInternalServerError, "Ошибка сервера")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"user": updatedUser})
		default:
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		}
	})

	mux.HandleFunc("/api/handle/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		if !handleCheckLimiter.Allow(fmt.Sprintf("handle-check:%d", userID)) {
			writeError(w, http.StatusTooManyRequests, "Слишком много запросов")
			return
		}
		available, reason := checkHandleAvailability(db, r.URL.Query().Get("handle"))
		writeJSON(w, http.StatusOK, map[string]any{"available": available, "reason": reason})
	})

	mux.HandleFunc("/api/profile/handle", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		if !handleUpdateLimiter.Allow(fmt.Sprintf("handle-put:%d", userID)) {
			writeError(w, http.StatusTooManyRequests, "Слишком много попыток")
			return
		}
		var req updateHandleRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "Некорректный JSON")
			return
		}
		u, err := setUserHandle(db, userID, req.Handle)
		if err != nil {
			handleAccountError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"user": u})
	})

	mux.HandleFunc("/api/settings", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		s, err := loadUserSettings(db, userID)
		if err != nil {
			handleAccountError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"settings": s})
	})

	mux.HandleFunc("/api/settings/notifications", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		var req updateNotificationsRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "Некорректный JSON")
			return
		}
		if err := updateNotificationSettings(db, userID, req); err != nil {
			handleAccountError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/api/settings/privacy", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		var req updatePrivacyRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "Некорректный JSON")
			return
		}
		if err := updatePrivacySettings(db, userID, req); err != nil {
			handleAccountError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/api/settings/appearance", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		var req updateAppearanceRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "Некорректный JSON")
			return
		}
		if err := updateAppearanceSettings(db, userID, req); err != nil {
			handleAccountError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/api/account/email", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		if !accountChangeLimiter.Allow(fmt.Sprintf("account-email:%d", userID)) {
			writeError(w, http.StatusTooManyRequests, "Слишком много попыток")
			return
		}
		var req changeEmailRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "Некорректный JSON")
			return
		}
		if err := changeUserEmail(db, userID, req.Email, req.CurrentPassword); err != nil {
			handleAccountError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/api/account/password", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		if !accountChangeLimiter.Allow(fmt.Sprintf("account-password:%d", userID)) {
			writeError(w, http.StatusTooManyRequests, "Слишком много попыток")
			return
		}
		var req changePasswordRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "Некорректный JSON")
			return
		}
		if err := changeUserPassword(db, sessions, userID, req.CurrentPassword, req.NewPassword, tokenFromRequest(r)); err != nil {
			handleAccountError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/api/account", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		if !accountDeleteLimiter.Allow(fmt.Sprintf("account-delete:%d", userID)) {
			writeError(w, http.StatusTooManyRequests, "Слишком много попыток")
			return
		}
		var req deleteAccountRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "Некорректный JSON")
			return
		}
		if err := deleteUserAccount(db, sessions, userID, req.CurrentPassword, req.Confirmation); err != nil {
			handleAccountError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/api/friends", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}

		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}

		friends, err := listFriends(db, userID)
		if err != nil {
			log.Printf("list friends error: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"friends": friends})
	})

	mux.HandleFunc("/api/friends/requests/incoming", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		requests, err := listIncomingFriendRequests(db, userID)
		if err != nil {
			log.Printf("list incoming friend requests error: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"requests": requests})
	})

	mux.HandleFunc("/api/friends/requests/outgoing", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		requests, err := listOutgoingFriendRequests(db, userID)
		if err != nil {
			log.Printf("list outgoing friend requests error: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"requests": requests})
	})

	mux.HandleFunc("/api/friends/candidates", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		candidates, err := listFriendCandidates(db, userID, query)
		if err != nil {
			log.Printf("list friend candidates error: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"users": candidates})
	})

	mux.HandleFunc("/api/friends/request-by-name", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		var req struct {
			Name string `json:"name"`
		}
		if err := decodeJSON(w, r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "Некорректный JSON")
			return
		}
		targetPublicID, err := findUserPublicIDByNameOrEmail(db, userID, req.Name)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusNotFound, "Пользователь не найден")
				return
			}
			if errors.Is(err, errValidation) {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			log.Printf("find user by name/email error: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}
		if err := createFriendRequest(db, userID, targetPublicID); err != nil {
			handleFriendActionError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/api/friends/request-by-handle", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		var req struct {
			Handle string `json:"handle"`
		}
		if err := decodeJSON(w, r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "Некорректный JSON")
			return
		}
		var targetPublicID string
		if err := db.QueryRow(`SELECT public_id FROM users WHERE LOWER(handle)=LOWER($1) AND is_deleted = FALSE`, strings.TrimPrefix(strings.TrimSpace(req.Handle), "@")).Scan(&targetPublicID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusNotFound, "Пользователь не найден")
				return
			}
			handleAccountError(w, err)
			return
		}
		if err := createFriendRequest(db, userID, targetPublicID); err != nil {
			handleFriendActionError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/api/friends/", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/api/friends/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			writeError(w, http.StatusBadRequest, "Некорректный путь")
			return
		}

		switch {
		case r.Method == http.MethodPost && len(parts) == 2 && parts[0] == "request":
			if err := createFriendRequest(db, userID, parts[1]); err != nil {
				handleFriendActionError(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
		case r.Method == http.MethodPost && len(parts) == 2 && parts[0] == "accept":
			if err := acceptFriendRequest(db, userID, parts[1]); err != nil {
				handleFriendActionError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		case r.Method == http.MethodPost && len(parts) == 2 && parts[0] == "reject":
			if err := rejectFriendRequest(db, userID, parts[1]); err != nil {
				handleFriendActionError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		case r.Method == http.MethodPost && len(parts) == 2 && parts[0] == "cancel":
			if err := cancelFriendRequest(db, userID, parts[1]); err != nil {
				handleFriendActionError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		case r.Method == http.MethodDelete && len(parts) == 1:
			if err := removeFriend(db, userID, parts[0]); err != nil {
				handleFriendActionError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		default:
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		}
	})

	mux.HandleFunc("/api/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		_, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}

		// Лимит 25 МБ
		const maxUploadSize = 25 * 1024 * 1024
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			writeError(w, http.StatusBadRequest, "Файл слишком большой или повреждён")
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			writeError(w, http.StatusBadRequest, "Файл не найден в запросе")
			return
		}
		defer file.Close()

		// Проверка расширения
		ext := strings.ToLower(filepath.Ext(header.Filename))
		blocked := map[string]bool{
			".exe": true, ".bat": true, ".cmd": true, ".sh": true,
			".html": true, ".htm": true, ".js": true, ".php": true,
			".jsp": true, ".dll": true, ".scr": true, ".com": true,
		}
		if blocked[ext] {
			writeError(w, http.StatusUnsupportedMediaType, "Тип файла не поддерживается")
			return
		}
		if ext == "" {
			ext = ".bin"
		}
		if len(ext) > 10 {
			writeError(w, http.StatusBadRequest, "Слишком длинное расширение файла")
			return
		}

		// Генерируем случайное имя
		randBytes := make([]byte, 16)
		if _, err := rand.Read(randBytes); err != nil {
			writeError(w, http.StatusInternalServerError, "Ошибка генерации имени")
			return
		}
		randName := hex.EncodeToString(randBytes) + ext

		// Структура: /data/uploads/YYYY/MM/<random>.<ext>
		now := time.Now()
		yearMonth := fmt.Sprintf("%04d/%02d", now.Year(), int(now.Month()))
		dir := filepath.Join("/data/uploads", yearMonth)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("upload: mkdir %s failed: %v", dir, err)
			writeError(w, http.StatusInternalServerError, "Не удалось создать директорию")
			return
		}
		fullPath := filepath.Join(dir, randName)

		out, err := os.Create(fullPath)
		if err != nil {
			log.Printf("upload: create %s failed: %v", fullPath, err)
			writeError(w, http.StatusInternalServerError, "Не удалось создать файл")
			return
		}
		defer out.Close()

		written, err := io.Copy(out, file)
		if err != nil {
			log.Printf("upload: copy failed: %v", err)
			_ = os.Remove(fullPath)
			writeError(w, http.StatusInternalServerError, "Не удалось сохранить файл")
			return
		}

		// Определяем MIME-type из заголовков формы (или fallback)
		mimeType := header.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		// URL для клиента
		url := fmt.Sprintf("/uploads/%s/%s", yearMonth, randName)

		writeJSON(w, http.StatusOK, map[string]any{
			"url":  url,
			"name": header.Filename,
			"size": written,
			"type": mimeType,
		})
	})

	chatPresence := newChatPresenceStore()
	chatMessageRateLimiter := newIPRateLimiter(60, time.Minute)
	chatTypingRateLimiter := newIPRateLimiter(20, time.Minute)
	chatCreateRateLimiter := newIPRateLimiter(10, time.Minute)

	mux.HandleFunc("/api/chat/conversations", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		chatPresence.touch(userID)
		filter := strings.TrimSpace(r.URL.Query().Get("filter"))
		if filter == "" {
			filter = "all"
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		items, err := listConversations(db, userID, filter, r.URL.Query().Get("q"), limit)
		if err != nil {
			handleChatError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"conversations": items})
	})

	mux.HandleFunc("/api/chat/conversations/direct", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		chatPresence.touch(userID)
		if !chatCreateRateLimiter.Allow(fmt.Sprintf("chat-direct:%d", userID)) {
			writeError(w, http.StatusTooManyRequests, "Слишком часто")
			return
		}
		var req createDirectConversationRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "Некорректный JSON")
			return
		}
		targetID, err := getUserIDByPublicID(db, req.UserPublicID)
		if err != nil {
			handleChatError(w, err)
			return
		}
		otherSettings, err := loadUserSettings(db, targetID)
		if err != nil {
			otherSettings = userSettings{PrivacyWhoCanMessage: "all"}
		}
		if !canViewerMessageOwner(db, userID, targetID, otherSettings.PrivacyWhoCanMessage) {
			writeError(w, http.StatusForbidden, "Этот пользователь ограничил приём сообщений")
			return
		}
		item, err := findOrCreateDirectConversation(db, userID, req.UserPublicID)
		if err != nil {
			handleChatError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"conversation": item})
	})

	mux.HandleFunc("/api/chat/conversations/group", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		chatPresence.touch(userID)
		if !chatCreateRateLimiter.Allow(fmt.Sprintf("chat-group:%d", userID)) {
			writeError(w, http.StatusTooManyRequests, "Слишком часто")
			return
		}
		var req createGroupConversationRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "Некорректный JSON")
			return
		}
		item, err := createGroupConversation(db, userID, req)
		if err != nil {
			handleChatError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"conversation": item})
	})

	mux.HandleFunc("/api/chat/conversations/", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		chatPresence.touch(userID)
		tail := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/chat/conversations/"), "/")
		parts := strings.Split(tail, "/")
		if len(parts) == 0 || parts[0] == "" {
			writeError(w, http.StatusBadRequest, "Некорректный путь")
			return
		}
		if parts[0] == "community" && len(parts) == 2 && r.Method == http.MethodPost {
			communityID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil || communityID <= 0 {
				writeError(w, http.StatusBadRequest, "Некорректный id сообщества")
				return
			}
			item, err := getOrCreateCommunityConversation(db, communityID, userID)
			if err != nil {
				handleChatError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"conversation": item})
			return
		}
		publicID := parts[0]
		if len(parts) == 1 && r.Method == http.MethodGet {
			item, err := getConversationByPublicID(db, userID, publicID)
			if err != nil {
				handleChatError(w, err)
				return
			}
			if item.Type == "direct" {
				otherID, _ := getUserIDByPublicID(db, item.OtherPublicID)
				item.IsOnline = chatPresence.isOnline(otherID)
			}
			writeJSON(w, http.StatusOK, map[string]any{"conversation": item})
			return
		}
		if len(parts) == 2 && parts[1] == "messages" {
			switch r.Method {
			case http.MethodGet:
				limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
				before, _ := strconv.ParseInt(r.URL.Query().Get("before_id"), 10, 64)
				msgs, err := listMessages(db, userID, publicID, limit, before)
				if err != nil {
					handleChatError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"messages": msgs})
			case http.MethodPost:
				if !chatMessageRateLimiter.Allow(fmt.Sprintf("chat-msg:%d", userID)) {
					writeError(w, http.StatusTooManyRequests, "Слишком часто")
					return
				}
				var req sendMessageRequest
				if err := decodeJSON(w, r, &req); err != nil {
					writeError(w, http.StatusBadRequest, "Некорректный JSON")
					return
				}
				msg, err := sendMessage(db, userID, publicID, req)
				if err != nil {
					handleChatError(w, err)
					return
				}
				writeJSON(w, http.StatusCreated, map[string]any{"message": msg})
			default:
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			}
			return
		}
		if len(parts) == 2 && parts[1] == "typing" {
			if r.Method == http.MethodPost {
				if !chatTypingRateLimiter.Allow(fmt.Sprintf("chat-typing:%d", userID)) {
					writeError(w, http.StatusTooManyRequests, "Слишком часто")
					return
				}
				var req typingRequest
				if err := decodeJSON(w, r, &req); err != nil {
					writeError(w, http.StatusBadRequest, "Некорректный JSON")
					return
				}
				if err := setTyping(db, userID, publicID, req.IsTyping); err != nil {
					handleChatError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
				return
			}
			if r.Method == http.MethodGet {
				items, err := listTyping(db, userID, publicID)
				if err != nil {
					handleChatError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"users": items})
				return
			}
		}
		if len(parts) == 2 && parts[1] == "read" && r.Method == http.MethodPost {
			var req markReadRequest
			if err := decodeJSON(w, r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "Некорректный JSON")
				return
			}
			if _, err := db.Exec(`UPDATE chat_participants p SET last_read_message_id=GREATEST(last_read_message_id,$1) FROM chat_conversations c WHERE c.id=p.conversation_id AND c.public_id=$2 AND p.user_id=$3`, req.MessageID, publicID, userID); err != nil {
				handleChatError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}
		if len(parts) == 2 && parts[1] == "pin" && r.Method == http.MethodPost {
			var req toggleBoolRequest
			if err := decodeJSON(w, r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "Некорректный JSON")
				return
			}
			_, err := db.Exec(`UPDATE chat_participants p SET pinned=$1 FROM chat_conversations c WHERE c.id=p.conversation_id AND c.public_id=$2 AND p.user_id=$3`, req.Pinned, publicID, userID)
			if err != nil {
				handleChatError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}
		if len(parts) == 2 && parts[1] == "mute" && r.Method == http.MethodPost {
			var req toggleBoolRequest
			if err := decodeJSON(w, r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "Некорректный JSON")
				return
			}
			_, err := db.Exec(`UPDATE chat_participants p SET muted=$1 FROM chat_conversations c WHERE c.id=p.conversation_id AND c.public_id=$2 AND p.user_id=$3`, req.Muted, publicID, userID)
			if err != nil {
				handleChatError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}
		if len(parts) == 2 && parts[1] == "leave" && r.Method == http.MethodPost {
			res, err := db.Exec(`DELETE FROM chat_participants p USING chat_conversations c WHERE c.id=p.conversation_id AND c.public_id=$1 AND p.user_id=$2 AND c.type IN ('group','community')`, publicID, userID)
			if err != nil {
				handleChatError(w, err)
				return
			}
			n, _ := res.RowsAffected()
			if n == 0 {
				writeError(w, http.StatusBadRequest, "Нельзя выйти из direct-диалога")
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}
		if len(parts) == 2 && parts[1] == "search" && r.Method == http.MethodGet {
			items, err := searchInConversation(db, userID, publicID, r.URL.Query().Get("q"))
			if err != nil {
				handleChatError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"messages": items})
			return
		}
		writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
	})

	mux.HandleFunc("/api/chat/messages/", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		chatPresence.touch(userID)
		idStr := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/chat/messages/"), "/")
		msgID, _ := strconv.ParseInt(idStr, 10, 64)
		if msgID == 0 {
			writeError(w, http.StatusBadRequest, "Некорректный id")
			return
		}
		if r.Method == http.MethodDelete {
			_, err := db.Exec(`UPDATE chat_messages SET is_deleted=TRUE,content='' WHERE id=$1 AND author_id=$2`, msgID, userID)
			if err != nil {
				handleChatError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}
		if r.Method == http.MethodPatch {
			var req editMessageRequest
			if err := decodeJSON(w, r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "Некорректный JSON")
				return
			}
			_, err := db.Exec(`UPDATE chat_messages SET content=$1,is_edited=TRUE,edited_at=NOW() WHERE id=$2 AND author_id=$3 AND created_at >= NOW()-INTERVAL '24 hours'`, strings.TrimSpace(req.Content), msgID, userID)
			if err != nil {
				handleChatError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}
		writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
	})

	mux.HandleFunc("/api/chat/presence/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		_, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		pid := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/chat/presence/"), "/")
		uid, err := getUserIDByPublicID(db, pid)
		if err != nil {
			handleChatError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"is_online": chatPresence.isOnline(uid)})
	})
	mux.HandleFunc("/api/communities", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			authUserID, hasAuth := optionalAuthenticatedUserID(r, sessions)
			communities, err := listCommunities(db, authUserID, hasAuth, communityListFilters{
				Privacy:  r.URL.Query().Get("privacy"),
				Category: r.URL.Query().Get("category"),
				Region:   r.URL.Query().Get("region"),
				Query:    r.URL.Query().Get("q"),
			})
			if err != nil {
				handleCommunityError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"communities": communities})
		case http.MethodPost:
			userID, ok := authenticatedUserID(w, r, sessions)
			if !ok {
				return
			}
			if !communityCreateRateLimiter.Allow(fmt.Sprintf("community-create:%d", userID)) {
				writeError(w, http.StatusTooManyRequests, "Слишком много созданий сообществ, попробуйте позже")
				return
			}

			var req createCommunityRequest
			if err := decodeJSON(w, r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "Некорректный JSON")
				return
			}

			created, err := createCommunity(db, userID, req)
			if err != nil {
				handleCommunityError(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, map[string]any{"community": created})
		default:
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		}
	})

	mux.HandleFunc("/api/communities/", func(w http.ResponseWriter, r *http.Request) {
		tail := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/communities/"), "/")
		if tail == "" {
			writeError(w, http.StatusNotFound, "Не найдено")
			return
		}
		parts := strings.Split(tail, "/")
		communityID, ok := parseCommunityID(parts[0])
		if !ok {
			writeError(w, http.StatusBadRequest, "Некорректный id сообщества")
			return
		}
		authUserID, hasAuth := optionalAuthenticatedUserID(r, sessions)
		if len(parts) == 1 {
			switch r.Method {
			case http.MethodGet:
				item, err := getCommunityByID(db, communityID, authUserID, hasAuth)
				if err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"community": item})
			case http.MethodPatch:
				userID, ok := authenticatedUserID(w, r, sessions)
				if !ok {
					return
				}
				var req updateCommunityRequest
				if err := decodeJSON(w, r, &req); err != nil {
					writeError(w, http.StatusBadRequest, "Некорректный JSON")
					return
				}
				item, err := updateCommunity(db, communityID, userID, req)
				if err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"community": item})
			case http.MethodDelete:
				userID, ok := authenticatedUserID(w, r, sessions)
				if !ok {
					return
				}
				if err := deleteCommunity(db, communityID, userID); err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			default:
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			}
			return
		}
		switch parts[1] {
		case "join":
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
				return
			}
			userID, ok := authenticatedUserID(w, r, sessions)
			if !ok {
				return
			}
			if err := joinCommunity(db, communityID, userID); err != nil {
				handleCommunityError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		case "leave":
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
				return
			}
			userID, ok := authenticatedUserID(w, r, sessions)
			if !ok {
				return
			}
			if err := leaveCommunity(db, communityID, userID); err != nil {
				handleCommunityError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		case "join-requests":
			userID, ok := authenticatedUserID(w, r, sessions)
			if !ok {
				return
			}
			if len(parts) == 2 && r.Method == http.MethodPost {
				if !communityJoinRequestRateLimiter.Allow(fmt.Sprintf("community-join-request:%d", userID)) {
					writeError(w, http.StatusTooManyRequests, "Слишком много заявок, попробуйте позже")
					return
				}
				var body joinRequestBody
				if err := decodeJSON(w, r, &body); err != nil {
					writeError(w, http.StatusBadRequest, "Некорректный JSON")
					return
				}
				if err := createJoinRequest(db, communityID, userID, body.Message); err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
				return
			}
			if len(parts) == 2 && r.Method == http.MethodGet {
				items, err := listJoinRequests(db, communityID, userID)
				if err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"requests": items})
				return
			}
			if len(parts) == 3 && parts[2] == "mine" && r.Method == http.MethodDelete {
				if err := cancelMyJoinRequest(db, communityID, userID); err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
				return
			}
			if len(parts) == 4 && r.Method == http.MethodPost && (parts[3] == "approve" || parts[3] == "reject") {
				requestID, ok := parseCommunityID(parts[2])
				if !ok {
					writeError(w, http.StatusBadRequest, "Некорректный id заявки")
					return
				}
				var err error
				if parts[3] == "approve" {
					err = approveJoinRequest(db, communityID, requestID, userID)
				} else {
					err = rejectJoinRequest(db, communityID, requestID, userID)
				}
				if err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
				return
			}
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		case "members":
			if len(parts) == 2 && r.Method == http.MethodGet {
				limit := parseLimit(r.URL.Query().Get("limit"), 50, 200)
				offset := parseLimit(r.URL.Query().Get("offset"), 0, 100000)
				items, err := listCommunityMembers(db, communityID, authUserID, hasAuth, limit, offset)
				if err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"members": items})
				return
			}
			userID, ok := authenticatedUserID(w, r, sessions)
			if !ok {
				return
			}
			if len(parts) == 4 && parts[3] == "role" && r.Method == http.MethodPut {
				var req changeRoleRequest
				if err := decodeJSON(w, r, &req); err != nil {
					writeError(w, http.StatusBadRequest, "Некорректный JSON")
					return
				}
				if err := changeCommunityRole(db, communityID, parts[2], req.Role, userID); err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
				return
			}
			if len(parts) == 3 && r.Method == http.MethodDelete {
				if err := kickCommunityMember(db, communityID, parts[2], userID); err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
				return
			}
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		case "posts":
			if len(parts) == 2 && r.Method == http.MethodGet {
				limit := parseLimit(r.URL.Query().Get("limit"), 20, 100)
				beforeID := parseIDOrZero(r.URL.Query().Get("before_id"))
				items, nextCursor, err := listCommunityPosts(db, communityID, authUserID, hasAuth, limit, beforeID)
				if err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"posts": items, "next_cursor": nextCursor})
				return
			}
			userID, ok := authenticatedUserID(w, r, sessions)
			if !ok {
				return
			}
			if len(parts) == 2 && r.Method == http.MethodPost {
				if !communityPostRateLimiter.Allow(fmt.Sprintf("community-post:%d", userID)) {
					writeError(w, http.StatusTooManyRequests, "Слишком много публикаций в сообществах, попробуйте позже")
					return
				}
				var req createPostRequest
				if err := decodeJSON(w, r, &req); err != nil {
					writeError(w, http.StatusBadRequest, "Некорректный JSON")
					return
				}
				item, err := createCommunityPost(db, communityID, userID, req)
				if err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusCreated, map[string]any{"post": item})
				return
			}
			if len(parts) == 3 && r.Method == http.MethodDelete {
				if err := deleteCommunityPost(db, communityID, parts[2], userID); err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
				return
			}
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		default:
			writeError(w, http.StatusNotFound, "Не найдено")
		}
	})

	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			authUserID, hasAuth := optionalAuthenticatedUserID(r, sessions)
			items, nextCursor, err := listEvents(db, authUserID, hasAuth, eventFilterFromRequest(r))
			if err != nil {
				handleEventError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"events": items, "next_cursor": nextCursor})
		case http.MethodPost:
			userID, ok := authenticatedUserID(w, r, sessions)
			if !ok {
				return
			}
			if !eventCreateLimiter.Allow(fmt.Sprintf("event-create:%d", userID)) {
				writeError(w, http.StatusTooManyRequests, "Слишком много попыток")
				return
			}
			var req createEventRequest
			if err := decodeJSON(w, r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "Некорректный JSON")
				return
			}
			item, err := createEvent(db, userID, req)
			if err != nil {
				handleEventError(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, map[string]any{"event": item})
		default:
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		}
	})

	mux.HandleFunc("/api/events/calendar", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		year, _ := strconv.Atoi(r.URL.Query().Get("year"))
		month, _ := strconv.Atoi(r.URL.Query().Get("month"))
		if year < 2000 || year > 2100 || month < 1 || month > 12 {
			writeError(w, http.StatusBadRequest, "Некорректные параметры календаря")
			return
		}
		authUserID, hasAuth := optionalAuthenticatedUserID(r, sessions)
		items, err := listEventsByDate(db, year, month, authUserID, hasAuth)
		if err != nil {
			handleEventError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"events": items})
	})

	mux.HandleFunc("/api/events/", func(w http.ResponseWriter, r *http.Request) {
		rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/events/"), "/")
		parts := strings.Split(rest, "/")
		if rest == "" || len(parts) == 0 {
			writeError(w, http.StatusNotFound, "Не найдено")
			return
		}
		publicID := parts[0]
		if !isValidEventPublicID(publicID) {
			writeError(w, http.StatusBadRequest, "Некорректный id мероприятия")
			return
		}
		if len(parts) == 1 {
			switch r.Method {
			case http.MethodGet:
				authUserID, hasAuth := optionalAuthenticatedUserID(r, sessions)
				item, err := getEvent(db, publicID, authUserID, hasAuth)
				if err != nil {
					handleEventError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"event": item})
			case http.MethodPatch:
				userID, ok := authenticatedUserID(w, r, sessions)
				if !ok {
					return
				}
				if !eventPatchLimiter.Allow(fmt.Sprintf("event-patch:%d", userID)) {
					writeError(w, http.StatusTooManyRequests, "Слишком много попыток")
					return
				}
				var req updateEventRequest
				if err := decodeJSON(w, r, &req); err != nil {
					writeError(w, http.StatusBadRequest, "Некорректный JSON")
					return
				}
				item, err := updateEvent(db, userID, publicID, req)
				if err != nil {
					handleEventError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"event": item})
			case http.MethodDelete:
				userID, ok := authenticatedUserID(w, r, sessions)
				if !ok {
					return
				}
				if !eventDeleteLimiter.Allow(fmt.Sprintf("event-delete:%d", userID)) {
					writeError(w, http.StatusTooManyRequests, "Слишком много попыток")
					return
				}
				if err := deleteEvent(db, userID, publicID); err != nil {
					handleEventError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			default:
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			}
			return
		}

		switch parts[1] {
		case "register":
			userID, ok := authenticatedUserID(w, r, sessions)
			if !ok {
				return
			}
			if !eventRegisterLimiter.Allow(fmt.Sprintf("event-register:%d", userID)) {
				writeError(w, http.StatusTooManyRequests, "Слишком много попыток")
				return
			}
			if r.Method == http.MethodPost {
				var req registerToEventRequest
				_ = decodeJSON(w, r, &req)
				status, count, err := registerToEvent(db, userID, publicID, req.TicketType)
				if err != nil {
					handleEventError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"status": status, "registered_count": count})
				return
			}
			if r.Method == http.MethodDelete {
				count, err := cancelEventRegistration(db, userID, publicID)
				if err != nil {
					handleEventError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"status": "cancelled", "registered_count": count})
				return
			}
		case "save":
			userID, ok := authenticatedUserID(w, r, sessions)
			if !ok {
				return
			}
			if !eventSaveLimiter.Allow(fmt.Sprintf("event-save:%d", userID)) {
				writeError(w, http.StatusTooManyRequests, "Слишком много попыток")
				return
			}
			if r.Method == http.MethodPost {
				count, err := saveEvent(db, userID, publicID)
				if err != nil {
					handleEventError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"status": "saved", "saved_count": count})
				return
			}
			if r.Method == http.MethodDelete {
				count, err := unsaveEvent(db, userID, publicID)
				if err != nil {
					handleEventError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "saved_count": count})
				return
			}
		case "registrations":
			if r.Method != http.MethodGet {
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
				return
			}
			userID, ok := authenticatedUserID(w, r, sessions)
			if !ok {
				return
			}
			limit := parseLimit(r.URL.Query().Get("limit"), 20, 100)
			items, err := listEventRegistrations(db, userID, publicID, limit)
			if err != nil {
				handleEventError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"registrations": items})
			return
		}
		writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
	})

	mux.HandleFunc("/api/posts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			userID, ok := authenticatedUserID(w, r, sessions)
			if !ok {
				return
			}
			if !postRateLimiter.Allow(fmt.Sprintf("post:%d", userID)) {
				writeError(w, http.StatusTooManyRequests, "Слишком много публикаций, попробуйте позже")
				return
			}
			var req createPostRequest
			if err := decodeJSON(w, r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "Некорректный JSON")
				return
			}
			created, err := createPost(db, userID, req)
			if err != nil {
				handlePostActionError(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, map[string]any{"post": created})
		default:
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		}
	})

	mux.HandleFunc("/api/posts/", func(w http.ResponseWriter, r *http.Request) {
		publicID, ok := extractPathParam(r.URL.Path, "/api/posts/")
		if !ok {
			writeError(w, http.StatusNotFound, "Не найдено")
			return
		}
		if strings.HasSuffix(publicID, "/like") {
			postPublicID := strings.TrimSuffix(publicID, "/like")
			if !isValidPostPublicID(postPublicID) {
				writeError(w, http.StatusBadRequest, "Некорректный id поста")
				return
			}
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
				return
			}
			userID, hasAuth := authenticatedUserID(w, r, sessions)
			if !hasAuth {
				return
			}
			isLiked, likesCount, err := toggleLike(db, postPublicID, userID)
			if err != nil {
				handlePostActionError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"is_liked": isLiked, "likes_count": likesCount})
			return
		}
		if strings.HasSuffix(publicID, "/save") {
			postPublicID := strings.TrimSuffix(publicID, "/save")
			if !isValidPostPublicID(postPublicID) {
				writeError(w, http.StatusBadRequest, "Некорректный id поста")
				return
			}
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
				return
			}
			userID, hasAuth := authenticatedUserID(w, r, sessions)
			if !hasAuth {
				return
			}
			isSaved, savesCount, err := togglePostSave(db, postPublicID, userID)
			if err != nil {
				handlePostActionError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"is_saved": isSaved, "saves_count": savesCount})
			return
		}
		if strings.HasSuffix(publicID, "/repost") {
			postPublicID := strings.TrimSuffix(publicID, "/repost")
			if !isValidPostPublicID(postPublicID) {
				writeError(w, http.StatusBadRequest, "Некорректный id поста")
				return
			}
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
				return
			}
			userID, hasAuth := authenticatedUserID(w, r, sessions)
			if !hasAuth {
				return
			}
			var req struct {
				Comment string `json:"comment"`
			}
			_ = decodeJSON(w, r, &req)
			repost, err := createRepost(db, postPublicID, userID, strings.TrimSpace(req.Comment))
			if err != nil {
				handlePostActionError(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, map[string]any{"post": repost})
			return
		}
		if strings.HasSuffix(publicID, "/view") {
			postPublicID := strings.TrimSuffix(publicID, "/view")
			if !isValidPostPublicID(postPublicID) {
				writeError(w, http.StatusBadRequest, "Некорректный id поста")
				return
			}
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
				return
			}
			userID, _ := optionalAuthenticatedUserID(r, sessions)
			ipHash := hashIP(clientIP(r))
			counted, count, err := registerPostView(db, postPublicID, userID, ipHash)
			if err != nil {
				handlePostActionError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"counted": counted, "views_count": count})
			return
		}
		if strings.HasSuffix(publicID, "/comments") {
			postPublicID := strings.TrimSuffix(publicID, "/comments")
			if !isValidPostPublicID(postPublicID) {
				writeError(w, http.StatusBadRequest, "Некорректный id поста")
				return
			}
			switch r.Method {
			case http.MethodGet:
				limit := parseLimit(r.URL.Query().Get("limit"), 50, 200)
				comments, err := listComments(db, postPublicID, limit)
				if err != nil {
					handlePostActionError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"comments": comments})
			case http.MethodPost:
				userID, hasAuth := authenticatedUserID(w, r, sessions)
				if !hasAuth {
					return
				}
				if !commentRateLimiter.Allow(fmt.Sprintf("comment:%d", userID)) {
					writeError(w, http.StatusTooManyRequests, "Слишком много комментариев, попробуйте позже")
					return
				}
				var req createCommentRequest
				if err := decodeJSON(w, r, &req); err != nil {
					writeError(w, http.StatusBadRequest, "Некорректный JSON")
					return
				}
				comment, err := createComment(db, postPublicID, userID, req)
				if err != nil {
					handlePostActionError(w, err)
					return
				}
				writeJSON(w, http.StatusCreated, map[string]any{"comment": comment})
			default:
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			}
			return
		}
		if !isValidPostPublicID(publicID) {
			writeError(w, http.StatusBadRequest, "Некорректный id поста")
			return
		}
		switch r.Method {
		case http.MethodGet:
			authUserID, hasAuth := optionalAuthenticatedUserID(r, sessions)
			item, err := getPostByID(db, publicID, authUserID, hasAuth, false)
			if err != nil {
				handlePostActionError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"post": item})
		case http.MethodDelete:
			userID, hasAuth := authenticatedUserID(w, r, sessions)
			if !hasAuth {
				return
			}
			if err := softDeletePost(db, publicID, userID); err != nil {
				handlePostActionError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		default:
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		}
	})

	mux.HandleFunc("/api/feed", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		authUserID, hasAuth := optionalAuthenticatedUserID(r, sessions)
		limit := parseLimit(r.URL.Query().Get("limit"), 20, 100)
		beforeID := parseIDOrZero(r.URL.Query().Get("before_id"))
		postType := strings.TrimSpace(r.URL.Query().Get("type"))
		posts, nextCursor, err := listFeed(db, authUserID, hasAuth, limit, beforeID, postType)
		if err != nil {
			handlePostActionError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"posts": posts, "next_cursor": nextCursor})
	})

	mux.HandleFunc("/api/posts/top", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}

		period := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("period")))
		if period != "day" && period != "week" {
			period = "week"
		}

		limit := 5
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n >= 1 && n <= 10 {
				limit = n
			}
		}

		authUserID, hasAuth := optionalAuthenticatedUserID(r, sessions)
		items, err := listTopPosts(db, authUserID, hasAuth, period, limit)
		if err != nil {
			log.Printf("listTopPosts failed: %v", err)
			writeError(w, http.StatusInternalServerError, "Не удалось получить топ постов")
			return
		}
		if items == nil {
			items = []post{}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"posts":  items,
			"period": period,
		})
	})

	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			viewerID, _ := optionalAuthenticatedUserID(r, sessions)
			filters := map[string]string{
				"category":   r.URL.Query().Get("category"),
				"status":     r.URL.Query().Get("status"),
				"owner_only": r.URL.Query().Get("owner_only"),
				"search":     r.URL.Query().Get("search"),
				"sort":       r.URL.Query().Get("sort"),
				"limit":      r.URL.Query().Get("limit"),
			}
			items, err := listProjects(db, viewerID, filters)
			if err != nil {
				log.Printf("listProjects: %v", err)
				writeError(w, http.StatusInternalServerError, "Не удалось получить проекты")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"projects": items})

		case http.MethodPost:
			userID, ok := authenticatedUserID(w, r, sessions)
			if !ok {
				return
			}
			var req createProjectRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "Некорректный JSON")
				return
			}
			p, err := createProject(db, userID, req)
			if err != nil {
				if errors.Is(err, errValidation) {
					writeError(w, http.StatusBadRequest, err.Error())
					return
				}
				log.Printf("createProject: %v", err)
				writeError(w, http.StatusInternalServerError, "Не удалось создать проект")
				return
			}
			writeJSON(w, http.StatusCreated, map[string]any{"project": p})

		default:
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		}
	})

	mux.HandleFunc("/api/projects/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			writeError(w, http.StatusBadRequest, "Не указан public_id")
			return
		}
		publicID := parts[0]

		if len(parts) == 1 {
			switch r.Method {
			case http.MethodGet:
				viewerID, _ := optionalAuthenticatedUserID(r, sessions)
				p, err := getProject(db, viewerID, publicID)
				if err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						writeError(w, http.StatusNotFound, "Проект не найден")
						return
					}
					log.Printf("getProject: %v", err)
					writeError(w, http.StatusInternalServerError, "Ошибка")
					return
				}
				members, _ := listProjectMembers(db, p.ID)
				writeJSON(w, http.StatusOK, map[string]any{"project": p, "members": members})

			case http.MethodPatch:
				userID, ok := authenticatedUserID(w, r, sessions)
				if !ok {
					return
				}
				var req updateProjectRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					writeError(w, http.StatusBadRequest, "Некорректный JSON")
					return
				}
				p, err := updateProject(db, userID, publicID, req)
				if err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						writeError(w, http.StatusNotFound, "Проект не найден")
						return
					}
					if errors.Is(err, errValidation) {
						writeError(w, http.StatusBadRequest, err.Error())
						return
					}
					if errors.Is(err, errForbidden) {
						writeError(w, http.StatusForbidden, err.Error())
						return
					}
					log.Printf("updateProject: %v", err)
					writeError(w, http.StatusInternalServerError, "Ошибка")
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"project": p})

			case http.MethodDelete:
				userID, ok := authenticatedUserID(w, r, sessions)
				if !ok {
					return
				}
				if err := deleteProject(db, userID, publicID); err != nil {
					if errors.Is(err, errForbidden) {
						writeError(w, http.StatusForbidden, err.Error())
						return
					}
					log.Printf("deleteProject: %v", err)
					writeError(w, http.StatusInternalServerError, "Ошибка")
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"deleted": true})

			default:
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			}
			return
		}

		if len(parts) == 2 && parts[1] == "members" && r.Method == http.MethodPost {
			userID, ok := authenticatedUserID(w, r, sessions)
			if !ok {
				return
			}
			var req addProjectMemberRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "Некорректный JSON")
				return
			}
			if err := addProjectMember(db, userID, publicID, req); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					writeError(w, http.StatusNotFound, "Проект или пользователь не найден")
					return
				}
				if errors.Is(err, errForbidden) {
					writeError(w, http.StatusForbidden, err.Error())
					return
				}
				log.Printf("addProjectMember: %v", err)
				writeError(w, http.StatusInternalServerError, "Ошибка")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"added": true})
			return
		}

		if len(parts) == 3 && parts[1] == "members" && r.Method == http.MethodDelete {
			userID, ok := authenticatedUserID(w, r, sessions)
			if !ok {
				return
			}
			if err := removeProjectMember(db, userID, publicID, parts[2]); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					writeError(w, http.StatusNotFound, "Не найден")
					return
				}
				if errors.Is(err, errForbidden) {
					writeError(w, http.StatusForbidden, err.Error())
					return
				}
				log.Printf("removeProjectMember: %v", err)
				writeError(w, http.StatusInternalServerError, "Ошибка")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"removed": true})
			return
		}

		writeError(w, http.StatusNotFound, "Маршрут не найден")
	})

	mux.HandleFunc("/api/notifications", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		limit := 30
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n >= 1 && n <= 100 {
				limit = n
			}
		}
		var beforeID int64
		if b := r.URL.Query().Get("before_id"); b != "" {
			if n, err := strconv.ParseInt(b, 10, 64); err == nil && n > 0 {
				beforeID = n
			}
		}
		onlyUnread := r.URL.Query().Get("unread") == "1"

		items, err := listNotifications(db, userID, limit, beforeID, onlyUnread)
		if err != nil {
			log.Printf("listNotifications: %v", err)
			writeError(w, http.StatusInternalServerError, "Не удалось получить уведомления")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"notifications": items})
	})

	mux.HandleFunc("/api/notifications/unread_count", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		n, err := countUnreadNotifications(db, userID)
		if err != nil {
			log.Printf("countUnreadNotifications: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"count": n})
	})

	mux.HandleFunc("/api/notifications/read_all", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		if err := markAllNotificationsRead(db, userID); err != nil {
			log.Printf("markAllNotificationsRead: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	mux.HandleFunc("/api/notifications/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/api/notifications/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) != 2 || parts[1] != "read" {
			writeError(w, http.StatusNotFound, "Маршрут не найден")
			return
		}
		notifID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil || notifID <= 0 {
			writeError(w, http.StatusBadRequest, "Некорректный id")
			return
		}
		if err := markNotificationRead(db, userID, notifID); err != nil {
			log.Printf("markNotificationRead: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	mux.HandleFunc("/api/users/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		if _, ok := authenticatedUserID(w, r, sessions); !ok {
			return
		}
		prefix := strings.TrimSpace(r.URL.Query().Get("prefix_handle"))
		if prefix != "" {
			items, err := searchUsersByHandle(db, prefix, parseLimit(r.URL.Query().Get("limit"), 8, 20))
			if err != nil {
				handleAccountError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"users": items})
			return
		}
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		items, err := db.Query(`
			SELECT public_id, full_name, COALESCE(handle,'')
			FROM users
			WHERE is_deleted = FALSE AND (LOWER(full_name) LIKE LOWER('%' || $1 || '%') OR LOWER(COALESCE(handle,'')) LIKE LOWER('%' || $1 || '%'))
			ORDER BY full_name
			LIMIT 20
		`, q)
		if err != nil {
			handleAccountError(w, err)
			return
		}
		defer items.Close()
		var out []friendCandidateDTO
		for items.Next() {
			var row friendCandidateDTO
			if err := items.Scan(&row.ID, &row.FullName, &row.Email); err != nil {
				handleAccountError(w, err)
				return
			}
			out = append(out, row)
		}
		writeJSON(w, http.StatusOK, map[string]any{"users": out})
	})

	mux.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if !searchLimiter.Allow(clientIP(r)) {
			writeError(w, http.StatusTooManyRequests, "too many requests")
			return
		}
		handleGlobalSearch(w, r, db, sessions)
	})

	mux.HandleFunc("/api/users/by-handle/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		if _, ok := authenticatedUserID(w, r, sessions); !ok {
			return
		}
		handle := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/users/by-handle/"))
		u, err := getUserByHandle(db, handle)
		if err != nil {
			handleAccountError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"user": u})
	})

	mux.HandleFunc("/api/mentions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		unread := r.URL.Query().Get("unread") == "true"
		limit := parseLimit(r.URL.Query().Get("limit"), 20, 100)
		query := `SELECT id, source_type, source_id, preview, is_read, created_at FROM user_mentions WHERE mentioned_user_id=$1`
		if unread {
			query += ` AND is_read=FALSE`
		}
		query += ` ORDER BY created_at DESC LIMIT $2`
		rows, err := db.Query(query, userID, limit)
		if err != nil {
			handleAccountError(w, err)
			return
		}
		defer rows.Close()
		type mention struct {
			ID         int64     `json:"id"`
			SourceType string    `json:"source_type"`
			SourceID   int64     `json:"source_id"`
			Preview    string    `json:"preview"`
			IsRead     bool      `json:"is_read"`
			CreatedAt  time.Time `json:"created_at"`
		}
		var items []mention
		for rows.Next() {
			var m mention
			if err := rows.Scan(&m.ID, &m.SourceType, &m.SourceID, &m.Preview, &m.IsRead, &m.CreatedAt); err != nil {
				handleAccountError(w, err)
				return
			}
			items = append(items, m)
		}
		writeJSON(w, http.StatusOK, map[string]any{"mentions": items})
	})

	mux.HandleFunc("/api/mentions/read-all", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		_, err := db.Exec(`UPDATE user_mentions SET is_read=TRUE WHERE mentioned_user_id=$1 AND is_read=FALSE`, userID)
		if err != nil {
			handleAccountError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/api/mentions/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/read") {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		idPart := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/mentions/"), "/read")
		mentionID, err := strconv.ParseInt(strings.Trim(idPart, "/"), 10, 64)
		if err != nil || mentionID <= 0 {
			writeError(w, http.StatusBadRequest, "Некорректный id")
			return
		}
		_, err = db.Exec(`UPDATE user_mentions SET is_read=TRUE WHERE id=$1 AND mentioned_user_id=$2`, mentionID, userID)
		if err != nil {
			handleAccountError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/api/users/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/communities") {
			if r.Method != http.MethodGet {
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
				return
			}
			publicID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/users/"), "/communities")
			publicID = strings.TrimSpace(strings.Trim(publicID, "/"))
			if !isValidUserPublicID(publicID) {
				writeError(w, http.StatusBadRequest, "Некорректный id пользователя")
				return
			}
			authUserID, hasAuth := optionalAuthenticatedUserID(r, sessions)
			items, err := listUserCommunities(db, publicID, authUserID, hasAuth)
			if err != nil {
				handleCommunityError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"communities": items})
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/posts") {
			if r.Method != http.MethodGet {
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
				return
			}
			publicID := strings.TrimSpace(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/users/"), "/"))
			if !isValidUserPublicID(publicID) {
				writeError(w, http.StatusBadRequest, "Некорректный id пользователя")
				return
			}
			viewerID, _ := optionalAuthenticatedUserID(r, sessions)
			profile, err := getPublicUserProfile(db, publicID, viewerID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					writeError(w, http.StatusNotFound, "Пользователь не найден")
					return
				}
				log.Printf("get public user profile error: %v", err)
				writeError(w, http.StatusInternalServerError, "Ошибка сервера")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"user": profile})
			return
		}
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		publicID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/users/"), "/posts")
		publicID = strings.TrimSpace(strings.TrimSuffix(publicID, "/"))
		if !isValidUserPublicID(publicID) {
			writeError(w, http.StatusBadRequest, "Некорректный id пользователя")
			return
		}
		authUserID, hasAuth := optionalAuthenticatedUserID(r, sessions)
		limit := parseLimit(r.URL.Query().Get("limit"), 20, 100)
		beforeID := parseIDOrZero(r.URL.Query().Get("before_id"))
		posts, nextCursor, err := listUserPosts(db, publicID, authUserID, hasAuth, limit, beforeID)
		if err != nil {
			handlePostActionError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"posts": posts, "next_cursor": nextCursor})
	})

	mux.HandleFunc("/api/news/", func(w http.ResponseWriter, r *http.Request) {
		suffix := strings.TrimPrefix(r.URL.Path, "/api/news/")
		if suffix == "" || strings.Contains(suffix, "//") {
			writeError(w, http.StatusNotFound, "Не найдено")
			return
		}
		r.URL.Path = "/api/posts/" + suffix
		mux.ServeHTTP(w, r)
	})

	// Раздача загруженных файлов
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("/data/uploads"))))

	mux.Handle("/", staticCacheControl(staticSecurity(injectHTML(http.FileServer(http.Dir("./web"))))))

	addr := ":8080"
	handler := gzipMiddleware(accessLog(securityHeaders(mux)))
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	log.Printf("Server started at http://localhost%s", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func initDBFromEnv() (*sql.DB, error) {
	databaseURL, err := resolveDatabaseURL()
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres db: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := waitForDB(db, 30*time.Second); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := ensureSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func resolveDatabaseURL() (string, error) {
	if directURL := strings.TrimSpace(os.Getenv("DATABASE_URL")); directURL != "" {
		return directURL, nil
	}

	return "", fmt.Errorf("DATABASE_URL is required")
}

func waitForDB(db *sql.DB, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		if err := db.PingContext(ctx); err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("database ping timeout after %s: %w", timeout, ctx.Err())
		case <-ticker.C:
		}
	}
}

func ensureSchema(db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    public_id TEXT UNIQUE,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    full_name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS public_id TEXT,
    ADD COLUMN IF NOT EXISTS position TEXT,
    ADD COLUMN IF NOT EXISTS company_name TEXT,
    ADD COLUMN IF NOT EXISTS bio TEXT,
    ADD COLUMN IF NOT EXISTS phone TEXT,
    ADD COLUMN IF NOT EXISTS location TEXT,
    ADD COLUMN IF NOT EXISTS city TEXT,
    ADD COLUMN IF NOT EXISTS avatar_url TEXT;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS handle TEXT,
    ADD COLUMN IF NOT EXISTS handle_changed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_password_change_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS locale TEXT NOT NULL DEFAULT 'ru',
    ADD COLUMN IF NOT EXISTS timezone TEXT NOT NULL DEFAULT 'Europe/Moscow',
    ADD COLUMN IF NOT EXISTS date_format TEXT NOT NULL DEFAULT 'DD.MM.YYYY',
    ADD COLUMN IF NOT EXISTS theme TEXT NOT NULL DEFAULT 'light',
    ADD COLUMN IF NOT EXISTS layout_mode TEXT NOT NULL DEFAULT 'normal',
    ADD COLUMN IF NOT EXISTS compact_feed BOOLEAN NOT NULL DEFAULT FALSE;

CREATE UNIQUE INDEX IF NOT EXISTS users_handle_lower_uniq_idx
    ON users (LOWER(handle)) WHERE handle IS NOT NULL AND handle <> '';

CREATE INDEX IF NOT EXISTS users_handle_prefix_idx
    ON users (LOWER(handle) text_pattern_ops) WHERE handle IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS users_public_id_uniq_idx ON users(public_id);

-- Миграция существующих users без public_id должна выполняться вручную.

CREATE TABLE IF NOT EXISTS sessions (
    token_hash TEXT PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '30 days')
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS friend_requests (
    id BIGSERIAL PRIMARY KEY,
    requester_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    addressee_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (requester_id <> addressee_id),
    CHECK (status IN ('pending', 'accepted', 'rejected', 'canceled'))
);

DO $$
BEGIN
    ALTER TABLE friend_requests DROP CONSTRAINT IF EXISTS friend_requests_status_check;
    ALTER TABLE friend_requests ADD CONSTRAINT friend_requests_status_check
      CHECK (status IN ('pending', 'accepted', 'rejected', 'canceled', 'unfriended'));
END$$;

CREATE UNIQUE INDEX IF NOT EXISTS friend_requests_pair_uniq_idx
ON friend_requests(requester_id, addressee_id);

CREATE INDEX IF NOT EXISTS friend_requests_addressee_status_idx
ON friend_requests(addressee_id, status);

CREATE INDEX IF NOT EXISTS friend_requests_requester_status_idx
ON friend_requests(requester_id, status);

CREATE TABLE IF NOT EXISTS communities (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    tags TEXT[] NOT NULL DEFAULT '{}',
    privacy_level TEXT NOT NULL DEFAULT 'open',
    creator_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (privacy_level IN ('open', 'closed'))
);

ALTER TABLE communities
    ADD COLUMN IF NOT EXISTS region TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS website TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS email TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS phone TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS avatar_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS cover_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS color TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS communities_created_at_idx
ON communities(created_at DESC);

CREATE TABLE IF NOT EXISTS community_members (
    community_id BIGINT NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (community_id, user_id)
);

ALTER TABLE community_members
    ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'member';

DO $$
BEGIN
  ALTER TABLE community_members DROP CONSTRAINT IF EXISTS community_members_role_check;
  ALTER TABLE community_members ADD CONSTRAINT community_members_role_check
    CHECK (role IN ('owner', 'admin', 'moderator', 'member'));
END$$;

CREATE UNIQUE INDEX IF NOT EXISTS community_members_owner_uniq_idx
    ON community_members(community_id) WHERE role = 'owner';

UPDATE community_members cm
SET role = 'owner'
FROM communities c
WHERE cm.community_id = c.id
  AND cm.user_id = c.creator_id
  AND cm.role = 'member';

INSERT INTO community_members (community_id, user_id, role)
SELECT c.id, c.creator_id, 'owner'
FROM communities c
WHERE NOT EXISTS (
    SELECT 1 FROM community_members cm
    WHERE cm.community_id = c.id AND cm.user_id = c.creator_id
)
ON CONFLICT (community_id, user_id) DO NOTHING;

CREATE INDEX IF NOT EXISTS community_members_user_idx
ON community_members(user_id);

CREATE TABLE IF NOT EXISTS community_join_requests (
    id BIGSERIAL PRIMARY KEY,
    community_id BIGINT NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message TEXT NOT NULL DEFAULT '' CHECK (char_length(message) <= 500),
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    decided_at TIMESTAMPTZ,
    decided_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    CHECK (status IN ('pending', 'approved', 'rejected', 'canceled'))
);

CREATE UNIQUE INDEX IF NOT EXISTS community_join_requests_pending_uniq_idx
    ON community_join_requests(community_id, user_id) WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS community_join_requests_community_idx
    ON community_join_requests(community_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS posts (
    id BIGSERIAL PRIMARY KEY,
    public_id TEXT UNIQUE NOT NULL,
    author_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL DEFAULT 'news',
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    cover_url TEXT,
    tags TEXT[] NOT NULL DEFAULT '{}',
    privacy_level TEXT NOT NULL DEFAULT 'public',
    likes_count INTEGER NOT NULL DEFAULT 0,
    comments_count INTEGER NOT NULL DEFAULT 0,
    views_count INTEGER NOT NULL DEFAULT 0,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (type IN ('news', 'project', 'note', 'case')),
    CHECK (privacy_level IN ('public', 'friends', 'private')),
    CHECK (char_length(title) BETWEEN 1 AND 200),
    CHECK (char_length(content) BETWEEN 1 AND 20000)
);

ALTER TABLE posts
    ADD COLUMN IF NOT EXISTS community_id BIGINT REFERENCES communities(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS reposted_from_id BIGINT REFERENCES posts(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS reposts_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS saves_count INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS posts_author_created_idx
    ON posts(author_id, created_at DESC) WHERE is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS posts_feed_idx
    ON posts(created_at DESC) WHERE is_deleted = FALSE AND privacy_level = 'public';

CREATE INDEX IF NOT EXISTS posts_type_created_idx
    ON posts(type, created_at DESC) WHERE is_deleted = FALSE AND privacy_level = 'public';

CREATE INDEX IF NOT EXISTS posts_community_created_idx
    ON posts(community_id, created_at DESC) WHERE is_deleted = FALSE AND community_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_posts_reposted_from
    ON posts(reposted_from_id) WHERE reposted_from_id IS NOT NULL;

-- ── Индексы для /api/stats и /api/posts/top (Спринт 3.5.1) ──

-- GIN-индекс на массив tags. Ускоряет:
--  • UNNEST(tags) GROUP BY tag в trend_today (/api/stats)
--  • будущие фильтры ленты по тегу (?tag=...)
--  • поиск постов по тегу через  tags @> ARRAY[...]
CREATE INDEX IF NOT EXISTS posts_tags_gin_idx
    ON posts USING GIN (tags) WHERE is_deleted = FALSE;

-- Expression-индекс на weighted score для /api/posts/top.
-- Формула: likes_count*3 + comments_count*2 + saves_count*5 + views_count*0.1
-- PostgreSQL может использовать его для ORDER BY по этому выражению
-- БЕЗ пересчёта на каждой строке.
CREATE INDEX IF NOT EXISTS posts_weighted_score_idx
    ON posts ((likes_count * 3 + comments_count * 2 + saves_count * 5 + views_count * 0.1) DESC, created_at DESC)
    WHERE is_deleted = FALSE AND privacy_level = 'public';

-- Composite индекс для пагинации ленты с курсором.
-- ORDER BY created_at DESC, id DESC + WHERE id < $beforeID попадает прямо в этот индекс.
CREATE INDEX IF NOT EXISTS posts_feed_pagination_idx
    ON posts (created_at DESC, id DESC) WHERE is_deleted = FALSE AND privacy_level = 'public';

CREATE TABLE IF NOT EXISTS post_likes (
    post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (post_id, user_id)
);

CREATE INDEX IF NOT EXISTS post_likes_user_idx ON post_likes(user_id);

CREATE TABLE IF NOT EXISTS post_saves (
    id BIGSERIAL PRIMARY KEY,
    post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (post_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_post_saves_user_created
    ON post_saves(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_post_saves_post
    ON post_saves(post_id);

CREATE TABLE IF NOT EXISTS post_views (
    id BIGSERIAL PRIMARY KEY,
    post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    ip_hash TEXT NOT NULL,
    viewed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_post_views_post_user
    ON post_views(post_id, user_id) WHERE user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_post_views_post_iphash_recent
    ON post_views(post_id, ip_hash, viewed_at DESC);

CREATE TABLE IF NOT EXISTS post_comments (
    id BIGSERIAL PRIMARY KEY,
    post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    author_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id BIGINT REFERENCES post_comments(id) ON DELETE CASCADE,
    content TEXT NOT NULL CHECK (char_length(content) BETWEEN 1 AND 5000),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS post_comments_post_idx
    ON post_comments(post_id, created_at DESC) WHERE is_deleted = FALSE;

-- Мероприятия платформы
CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    public_id TEXT UNIQUE NOT NULL,
    organizer_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    community_id BIGINT REFERENCES communities(id) ON DELETE SET NULL,
    title TEXT NOT NULL CHECK (char_length(title) BETWEEN 3 AND 200),
    description TEXT NOT NULL DEFAULT '' CHECK (char_length(description) <= 10000),
    type TEXT NOT NULL DEFAULT 'webinar',
    format TEXT NOT NULL DEFAULT 'online',
    category TEXT NOT NULL DEFAULT '',
    city TEXT NOT NULL DEFAULT '',
    address TEXT NOT NULL DEFAULT '',
    venue TEXT NOT NULL DEFAULT '',
    online_url TEXT NOT NULL DEFAULT '',
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    timezone TEXT NOT NULL DEFAULT 'Europe/Moscow',
    cover_url TEXT NOT NULL DEFAULT '',
    banner_color TEXT NOT NULL DEFAULT '',
    fee_cents INTEGER NOT NULL DEFAULT 0 CHECK (fee_cents >= 0),
    currency TEXT NOT NULL DEFAULT 'RUB',
    seats_total INTEGER NOT NULL DEFAULT 0 CHECK (seats_total >= 0),
    registered_count INTEGER NOT NULL DEFAULT 0,
    saved_count INTEGER NOT NULL DEFAULT 0,
    views_count INTEGER NOT NULL DEFAULT 0,
    tags TEXT[] NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'published',
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (type IN ('webinar', 'conference', 'workshop', 'networking', 'roundtable', 'meetup', 'other')),
    CHECK (format IN ('online', 'offline', 'hybrid')),
    CHECK (status IN ('draft', 'published', 'cancelled', 'finished')),
    CHECK (ends_at >= starts_at)
);

CREATE INDEX IF NOT EXISTS events_starts_at_idx
    ON events (starts_at) WHERE is_deleted = FALSE AND status = 'published';
CREATE INDEX IF NOT EXISTS events_organizer_idx
    ON events (organizer_id, starts_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS events_community_idx
    ON events (community_id, starts_at DESC) WHERE community_id IS NOT NULL AND is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS events_type_idx
    ON events (type, starts_at DESC) WHERE is_deleted = FALSE AND status = 'published';

CREATE TABLE IF NOT EXISTS event_registrations (
    event_id BIGINT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ticket_type TEXT NOT NULL DEFAULT 'standard',
    status TEXT NOT NULL DEFAULT 'confirmed',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id, user_id),
    CHECK (status IN ('confirmed', 'cancelled', 'waitlist'))
);

CREATE INDEX IF NOT EXISTS event_registrations_user_idx
    ON event_registrations (user_id, created_at DESC) WHERE status = 'confirmed';

CREATE TABLE IF NOT EXISTS event_saves (
    event_id BIGINT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id, user_id)
);

CREATE INDEX IF NOT EXISTS event_saves_user_idx
    ON event_saves (user_id, created_at DESC);

-- ── Проекты (Спринт 6.1) ──

CREATE TABLE IF NOT EXISTS projects (
    id BIGSERIAL PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL CHECK (char_length(title) BETWEEN 1 AND 200),
    description TEXT NOT NULL DEFAULT '' CHECK (char_length(description) <= 5000),
    category TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('planned', 'active', 'paused', 'done')),
    deadline TIMESTAMPTZ,
    budget BIGINT,
    cover_color TEXT NOT NULL DEFAULT '#1E8A4C',
    tags TEXT[] NOT NULL DEFAULT '{}',
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS projects_created_idx
    ON projects(created_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS projects_owner_idx
    ON projects(owner_id) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS projects_status_idx
    ON projects(status) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS projects_tags_gin_idx
    ON projects USING GIN(tags) WHERE is_deleted = FALSE;

CREATE TABLE IF NOT EXISTS project_members (
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (project_id, user_id)
);

CREATE INDEX IF NOT EXISTS project_members_user_idx
    ON project_members(user_id);

CREATE TABLE IF NOT EXISTS user_settings (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    notif_email BOOLEAN NOT NULL DEFAULT TRUE,
    notif_push BOOLEAN NOT NULL DEFAULT FALSE,
    notif_friend_requests BOOLEAN NOT NULL DEFAULT TRUE,
    notif_chat_messages BOOLEAN NOT NULL DEFAULT TRUE,
    notif_mentions BOOLEAN NOT NULL DEFAULT TRUE,
    notif_reactions BOOLEAN NOT NULL DEFAULT FALSE,
    notif_new_jobs BOOLEAN NOT NULL DEFAULT TRUE,
    notif_events BOOLEAN NOT NULL DEFAULT TRUE,
    notif_news_digest BOOLEAN NOT NULL DEFAULT FALSE,
    notif_platform_updates BOOLEAN NOT NULL DEFAULT TRUE,
    privacy_profile_private BOOLEAN NOT NULL DEFAULT FALSE,
    privacy_show_email BOOLEAN NOT NULL DEFAULT FALSE,
    privacy_show_phone BOOLEAN NOT NULL DEFAULT FALSE,
    privacy_discoverable BOOLEAN NOT NULL DEFAULT TRUE,
    privacy_show_online BOOLEAN NOT NULL DEFAULT TRUE,
    privacy_show_last_seen BOOLEAN NOT NULL DEFAULT FALSE,
    privacy_who_can_message TEXT NOT NULL DEFAULT 'all',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (privacy_who_can_message IN ('all', 'contacts', 'nobody'))
);

CREATE TABLE IF NOT EXISTS user_mentions (
    id BIGSERIAL PRIMARY KEY,
    mentioned_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    actor_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source_type TEXT NOT NULL,
    source_id BIGINT NOT NULL,
    preview TEXT NOT NULL DEFAULT '',
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (source_type IN ('post', 'comment', 'forum_message'))
);

CREATE INDEX IF NOT EXISTS user_mentions_user_unread_idx
    ON user_mentions (mentioned_user_id, is_read, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS user_mentions_source_uniq_idx
    ON user_mentions (source_type, source_id, mentioned_user_id);

-- Диалоги: прямые (direct, 1-на-1) и групповые (group)
CREATE TABLE IF NOT EXISTS chat_conversations (
    id BIGSERIAL PRIMARY KEY,
    public_id TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL DEFAULT 'direct',
    title TEXT NOT NULL DEFAULT '',
    avatar_url TEXT NOT NULL DEFAULT '',
    community_id BIGINT REFERENCES communities(id) ON DELETE CASCADE,
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    last_message_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (type IN ('direct', 'group', 'community'))
);

CREATE INDEX IF NOT EXISTS chat_conversations_community_idx
    ON chat_conversations(community_id) WHERE community_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS chat_participants (
    conversation_id BIGINT NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_read_message_id BIGINT NOT NULL DEFAULT 0,
    muted BOOLEAN NOT NULL DEFAULT FALSE,
    pinned BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (conversation_id, user_id),
    CHECK (role IN ('owner', 'admin', 'member'))
);

CREATE INDEX IF NOT EXISTS chat_participants_user_idx
    ON chat_participants(user_id);

CREATE TABLE IF NOT EXISTS chat_messages (
    id BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
    author_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL CHECK (char_length(content) BETWEEN 1 AND 8000),
    reply_to_id BIGINT REFERENCES chat_messages(id) ON DELETE SET NULL,
    is_edited BOOLEAN NOT NULL DEFAULT FALSE,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    edited_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS chat_messages_conversation_idx
    ON chat_messages(conversation_id, id DESC) WHERE is_deleted = FALSE;

-- Спринт 5: вложения в сообщениях чата.
-- Разрешаем пустой content если есть attachment.
ALTER TABLE chat_messages DROP CONSTRAINT IF EXISTS chat_messages_content_check;
ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS attachment_url TEXT NOT NULL DEFAULT '';
ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS attachment_name TEXT NOT NULL DEFAULT '';
ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS attachment_type TEXT NOT NULL DEFAULT '';
ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS attachment_size BIGINT NOT NULL DEFAULT 0;
-- Новый CHECK: либо есть текст, либо есть attachment.
ALTER TABLE chat_messages DROP CONSTRAINT IF EXISTS chat_messages_content_or_attachment_check;
ALTER TABLE chat_messages ADD CONSTRAINT chat_messages_content_or_attachment_check
    CHECK (char_length(content) BETWEEN 0 AND 8000 AND (char_length(content) > 0 OR char_length(attachment_url) > 0));

CREATE TABLE IF NOT EXISTS chat_typing (
    conversation_id BIGINT NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (conversation_id, user_id)
);

CREATE INDEX IF NOT EXISTS chat_typing_started_idx
    ON chat_typing(started_at);

-- ── Уведомления (Спринт 12.1) ──

CREATE TABLE IF NOT EXISTS notifications (
    id BIGSERIAL PRIMARY KEY,
    recipient_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    actor_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    type TEXT NOT NULL,
    source_type TEXT NOT NULL DEFAULT '',
    source_id BIGINT NOT NULL DEFAULT 0,
    source_public_id TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL CHECK (char_length(title) BETWEEN 1 AND 500),
    preview TEXT NOT NULL DEFAULT '' CHECK (char_length(preview) <= 1000),
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS notifications_recipient_unread_idx
    ON notifications (recipient_id, is_read, created_at DESC);

CREATE INDEX IF NOT EXISTS notifications_recipient_created_idx
    ON notifications (recipient_id, created_at DESC);

-- Идемпотентность: одно событие одного типа от одного актора по одному источнику = одна запись.
-- При попытке вставить дубль — ON CONFLICT DO NOTHING.
CREATE UNIQUE INDEX IF NOT EXISTS notifications_dedup_idx
    ON notifications (recipient_id, type, source_type, source_id, COALESCE(actor_id, 0));
`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	if err := migratePublicationSeedData(db); err != nil {
		log.Printf("WARN: migrate publication seed data failed: %v", err)
	}
	if err := migrateEventSeedData(db); err != nil {
		log.Printf("WARN: migrate event seed data failed: %v", err)
	}

	return nil
}

type seedPost struct {
	PublicID   string
	Title      string
	Content    string
	Tags       []string
	CoverURL   string
	CreatedAt  time.Time
	ViewsCount int
	LikesCount int
}

func migratePublicationSeedData(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const seedUserEmail = "seed.lastop@local"

	if _, err := tx.Exec(`
		UPDATE users SET public_id = NULL
		WHERE email = $1 AND public_id = 'usr_seed_lastop'
	`, seedUserEmail); err != nil {
		return err
	}

	var seedAuthorID int64
	if err := tx.QueryRow(`
		INSERT INTO users (public_id, first_name, last_name, full_name, email, password_hash, position, company_name, bio, avatar_url)
		VALUES ('u5eed1a5f0b', 'LASTOP', 'Digital', 'LASTOP Digital', $1, '$2a$10$H8hT4aSt3D8QdDkbx73Z9OBPFnfRTE8Yv5IadB1iKsFcfoTmya8Ie', 'Редакция LASTOP', 'LASTOP', 'Системный аккаунт для миграции публикаций', '')
		ON CONFLICT (email) DO UPDATE
		SET public_id = EXCLUDED.public_id,
		    full_name = EXCLUDED.full_name,
		    position = EXCLUDED.position,
		    company_name = EXCLUDED.company_name
		RETURNING id
	`, seedUserEmail).Scan(&seedAuthorID); err != nil {
		return err
	}

	now := time.Now().UTC()
	seeds := []seedPost{
		{
			PublicID:   "pst_seed_customs_2026",
			Title:      "ФТС обновила требования к таможенному декларированию импорта",
			Content:    "С 1 мая 2026 года вступают в силу изменения в порядке декларирования отдельных категорий товаров. Бизнесу рекомендуется заранее проверить шаблоны документов и обновить внутренние регламенты.",
			Tags:       []string{"таможня", "вэд", "регуляторика"},
			CreatedAt:  now.Add(-18 * time.Hour),
			ViewsCount: 1820,
			LikesCount: 47,
		},
		{
			PublicID:   "pst_seed_routes_2026",
			Title:      "Рынок фиксирует рост контейнерных перевозок на 18% в Q2 2026",
			Content:    "Участники отрасли связывают рост с развитием альтернативных международных маршрутов и повышением прозрачности цифрового трекинга грузов.",
			Tags:       []string{"логистика", "контейнеры", "аналитика"},
			CreatedAt:  now.Add(-48 * time.Hour),
			ViewsCount: 2400,
			LikesCount: 52,
		},
		{
			PublicID:   "pst_seed_tms_ai",
			Title:      "LASTOP Digital запустил модуль аналитики TMS с ИИ‑прогнозированием",
			Content:    "Новый модуль помогает предсказывать отклонения SLA и автоматически сигнализирует о рисках на отдельных участках цепочки поставок.",
			Tags:       []string{"технологии", "tms", "ai"},
			CreatedAt:  now.Add(-72 * time.Hour),
			ViewsCount: 940,
			LikesCount: 34,
		},
	}

	for _, item := range seeds {
		var pgTags pgtype.FlatArray[string] = item.Tags

		if _, err := tx.Exec(`
			INSERT INTO posts (public_id, author_id, type, title, content, tags, cover_url, privacy_level, views_count, likes_count, created_at, updated_at)
			VALUES ($1, $2, 'news', $3, $4, $5, NULLIF($6, ''), 'public', $7, $8, $9, NOW())
			ON CONFLICT (public_id) DO NOTHING
		`, item.PublicID, seedAuthorID, item.Title, item.Content, pgTags, item.CoverURL, item.ViewsCount, item.LikesCount, item.CreatedAt); err != nil {
			return err
		}
	}

	return tx.Commit()
}

var (
	errValidation         = errors.New("validation error")
	errEmailTaken         = errors.New("email taken")
	errInvalidCredentials = errors.New("invalid credentials")
	errConflict           = errors.New("conflict")
	errNotFound           = errors.New("not found")
	errForbidden          = errors.New("forbidden")
	errHandleTaken        = errors.New("handle taken")
	errAlreadyMember      = errors.New("already member")
	errRequestPending     = errors.New("request pending")

	handleRe        = regexp.MustCompile(`^[a-z0-9][a-z0-9_]{2,23}$`)
	mentionRe       = regexp.MustCompile(`(?i)@([a-z0-9_]{3,24})`)
	reservedHandles = map[string]struct{}{
		"admin": {}, "root": {}, "support": {}, "system": {}, "anonymous": {},
		"lastop": {}, "help": {}, "api": {}, "bot": {}, "null": {}, "undefined": {},
		"deleted": {}, "me": {}, "settings": {}, "profile": {}, "chat": {},
		"news": {}, "forum": {}, "jobs": {}, "events": {}, "communities": {},
		"companies": {}, "login": {}, "register": {}, "logout": {},
		"new": {}, "edit": {}, "delete": {},
	}
)

func createUser(db *sql.DB, req registerRequest) (user, error) {
	if strings.TrimSpace(req.FirstName) == "" {
		return user{}, fmt.Errorf("%w: имя обязательно", errValidation)
	}
	if strings.TrimSpace(req.LastName) == "" {
		return user{}, fmt.Errorf("%w: фамилия обязательна", errValidation)
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !isValidEmail(email) {
		return user{}, fmt.Errorf("%w: email некорректен", errValidation)
	}
	if len(req.Password) < 8 {
		return user{}, fmt.Errorf("%w: пароль должен быть не короче 8 символов", errValidation)
	}
	if !req.Terms {
		return user{}, fmt.Errorf("%w: необходимо принять условия", errValidation)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return user{}, fmt.Errorf("hash password: %w", err)
	}

	fullName := strings.TrimSpace(req.FullName)
	if fullName == "" {
		fullName = strings.TrimSpace(req.FirstName + " " + req.LastName)
	}

	for attempts := 0; attempts < 5; attempts++ {
		publicID, genErr := newPublicUserID()
		if genErr != nil {
			return user{}, fmt.Errorf("generate public_id: %w", genErr)
		}

		var created user
		err = db.QueryRow(`
			INSERT INTO users(public_id, first_name, last_name, full_name, email, password_hash)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, public_id, first_name, last_name, full_name, email,
				COALESCE(position, ''), COALESCE(company_name, ''), COALESCE(bio, ''),
				COALESCE(phone, ''), COALESCE(location, ''), COALESCE(city, ''), COALESCE(avatar_url, ''), COALESCE(handle, '')
		`, publicID, strings.TrimSpace(req.FirstName), strings.TrimSpace(req.LastName), fullName, email, string(hash)).
			Scan(&created.ID, &created.PublicID, &created.FirstName, &created.LastName, &created.FullName, &created.Email, &created.Position, &created.CompanyName, &created.Bio, &created.Phone, &created.Location, &created.City, &created.AvatarURL, &created.Handle)
		if err == nil {
			return created, nil
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" && pgErr.ConstraintName == "users_email_key" {
				return user{}, errEmailTaken
			}
			if pgErr.Code == "23505" {
				continue
			}
		}

		return user{}, err
	}

	return user{}, fmt.Errorf("failed to allocate public_id")
}

func loginUser(db *sql.DB, req loginRequest) (user, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !isValidEmail(email) {
		return user{}, fmt.Errorf("%w: email некорректен", errValidation)
	}
	if strings.TrimSpace(req.Password) == "" {
		return user{}, fmt.Errorf("%w: пароль обязателен", errValidation)
	}

	var u user
	var passwordHash string
	err := db.QueryRow(`
		SELECT id, public_id, first_name, last_name, full_name, email,
			COALESCE(position, ''), COALESCE(company_name, ''), COALESCE(bio, ''),
			COALESCE(phone, ''), COALESCE(location, ''), COALESCE(city, ''), COALESCE(avatar_url, ''), COALESCE(handle, ''),
			password_hash
		FROM users
		WHERE email = $1 AND is_deleted = FALSE
	`, email).Scan(&u.ID, &u.PublicID, &u.FirstName, &u.LastName, &u.FullName, &u.Email, &u.Position, &u.CompanyName, &u.Bio, &u.Phone, &u.Location, &u.City, &u.AvatarURL, &u.Handle, &passwordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user{}, errInvalidCredentials
		}
		return user{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return user{}, errInvalidCredentials
	}

	return u, nil
}

func getPublicUserProfile(db *sql.DB, publicID string, viewerID int64) (publicUserProfile, error) {
	var profile publicUserProfile
	var ownerID int64
	err := db.QueryRow(`
		SELECT id, public_id, first_name, last_name, full_name, email, COALESCE(handle, ''),
			COALESCE(avatar_url, ''), COALESCE(position, ''), COALESCE(company_name, ''),
			COALESCE(bio, ''), COALESCE(phone, ''), COALESCE(location, ''), COALESCE(city, '')
		FROM users
		WHERE public_id = $1 AND is_deleted = FALSE
	`, publicID).Scan(
		&ownerID,
		&profile.PublicID,
		&profile.FirstName,
		&profile.LastName,
		&profile.FullName,
		&profile.Email,
		&profile.Handle,
		&profile.AvatarURL,
		&profile.Position,
		&profile.CompanyName,
		&profile.Bio,
		&profile.Phone,
		&profile.Location,
		&profile.City,
	)
	if err != nil {
		return publicUserProfile{}, err
	}
	isSelf := viewerID > 0 && viewerID == ownerID
	profile.IsSelf = isSelf
	settings, err := loadUserSettings(db, ownerID)
	if err != nil {
		settings = userSettings{
			PrivacyProfilePrivate: false,
			PrivacyShowEmail:      false,
			PrivacyShowPhone:      false,
			PrivacyShowOnline:     true,
			PrivacyShowLastSeen:   false,
			PrivacyWhoCanMessage:  "all",
		}
	}
	profile.IsPrivate = settings.PrivacyProfilePrivate && !isSelf
	if profile.IsPrivate {
		profile.FirstName = ""
		profile.LastName = ""
		profile.Position = ""
		profile.CompanyName = ""
		profile.Bio = ""
		profile.Phone = ""
		profile.Email = ""
		profile.Location = ""
		profile.City = ""
		profile.CanMessage = false
		return profile, nil
	}
	if !isSelf && !settings.PrivacyShowEmail {
		profile.Email = ""
	}
	if !isSelf && !settings.PrivacyShowPhone {
		profile.Phone = ""
	}
	if isSelf || settings.PrivacyShowOnline {
		profile.IsOnline = hasActiveSession(db, ownerID)
	}
	if isSelf || settings.PrivacyShowLastSeen {
		var seen sql.NullTime
		if err := db.QueryRow(`SELECT MAX(last_seen_at) FROM sessions WHERE user_id=$1`, ownerID).Scan(&seen); err == nil && seen.Valid {
			formatted := seen.Time.UTC().Format(time.RFC3339)
			profile.LastSeenAt = &formatted
		}
	}
	profile.CanMessage = canViewerMessageOwner(db, viewerID, ownerID, settings.PrivacyWhoCanMessage)
	profile.FriendStatus = computeFriendStatus(db, viewerID, ownerID)
	return profile, nil
}

// computeFriendStatus возвращает статус отношения viewerID к ownerID:
//
//	"self"     — это сам юзер
//	"friends"  — заявка accepted
//	"outgoing" — viewer отправил заявку (pending)
//	"incoming" — owner отправил заявку viewer'у (pending)
//	"none"     — никаких отношений (или были rejected/canceled/unfriended)
//
// Если viewerID == 0 (не авторизован) — возвращает "none".
func computeFriendStatus(db *sql.DB, viewerID, ownerID int64) string {
	if viewerID == 0 {
		return "none"
	}
	if viewerID == ownerID {
		return "self"
	}
	var requesterID, addresseeID int64
	var status string
	err := db.QueryRow(`
		SELECT requester_id, addressee_id, status
		FROM friend_requests
		WHERE (requester_id = $1 AND addressee_id = $2)
		   OR (requester_id = $2 AND addressee_id = $1)
		LIMIT 1
	`, viewerID, ownerID).Scan(&requesterID, &addresseeID, &status)
	if err != nil {
		return "none" // нет записи или ошибка → none
	}
	switch status {
	case "accepted":
		return "friends"
	case "pending":
		if requesterID == viewerID {
			return "outgoing"
		}
		return "incoming"
	default:
		// rejected, canceled, unfriended → можно отправить новую
		return "none"
	}
}

func canViewerMessageOwner(db *sql.DB, viewerID, ownerID int64, whoCanMessage string) bool {
	if viewerID == 0 || viewerID == ownerID {
		return false
	}
	switch strings.TrimSpace(strings.ToLower(whoCanMessage)) {
	case "nobody":
		return false
	case "contacts":
		var hasContact bool
		err := db.QueryRow(`
			SELECT EXISTS(
				SELECT 1
				FROM chat_conversations cc
				JOIN chat_participants p1 ON p1.conversation_id = cc.id AND p1.user_id = $1
				JOIN chat_participants p2 ON p2.conversation_id = cc.id AND p2.user_id = $2
				WHERE cc.type = 'direct'
			)
		`, viewerID, ownerID).Scan(&hasContact)
		return err == nil && hasContact
	default:
		return true
	}
}

func hasActiveSession(db *sql.DB, userID int64) bool {
	var isOnline bool
	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM sessions WHERE user_id=$1 AND expires_at > NOW())`, userID).Scan(&isOnline)
	return err == nil && isOnline
}

func getUserByID(db *sql.DB, userID int64) (user, error) {
	var u user
	err := db.QueryRow(`
		SELECT id, public_id, first_name, last_name, full_name, email,
			COALESCE(position, ''), COALESCE(company_name, ''), COALESCE(bio, ''),
			COALESCE(phone, ''), COALESCE(location, ''), COALESCE(city, ''), COALESCE(avatar_url, ''), COALESCE(handle, '')
		FROM users
		WHERE id = $1 AND is_deleted = FALSE
	`, userID).Scan(&u.ID, &u.PublicID, &u.FirstName, &u.LastName, &u.FullName, &u.Email, &u.Position, &u.CompanyName, &u.Bio, &u.Phone, &u.Location, &u.City, &u.AvatarURL, &u.Handle)
	if err != nil {
		return user{}, err
	}

	return u, nil
}

func authenticatedUserID(w http.ResponseWriter, r *http.Request, sessions *sessionStore) (int64, bool) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(authHeader, "Bearer ") {
		writeError(w, http.StatusUnauthorized, "Требуется авторизация")
		return 0, false
	}
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if token == "" {
		writeError(w, http.StatusUnauthorized, "Требуется авторизация")
		return 0, false
	}
	userID, ok := sessions.getUserID(token)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Сессия не найдена")
		return 0, false
	}
	return userID, true
}

func optionalAuthenticatedUserID(r *http.Request, sessions *sessionStore) (int64, bool) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return 0, false
	}
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if token == "" {
		return 0, false
	}
	userID, ok := sessions.getUserID(token)
	if !ok {
		return 0, false
	}
	return userID, true
}

func tokenFromRequest(r *http.Request) string {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
}

// createNotification создаёт уведомление для recipient'а.
// Идемпотентно через unique-индекс: одно событие одного типа от одного актора
// по одному источнику = одна запись.
//
// Не пишет уведомление если:
//   - recipient == actor (юзер сам себе);
//   - recipient'а не существует;
//   - у recipient'а соответствующий notif-тогл выключен (для известных типов).
//
// Не возвращает ошибку если уведомление "не нужно создавать" — это нормальный путь.
func createNotification(db *sql.DB, p createNotificationParams) error {
	if p.RecipientID == 0 || strings.TrimSpace(p.Type) == "" || strings.TrimSpace(p.Title) == "" {
		return fmt.Errorf("createNotification: invalid params")
	}
	if p.RecipientID == p.ActorID && p.ActorID != 0 {
		return nil
	}

	var recipientExists bool
	if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND is_deleted = FALSE)`, p.RecipientID).Scan(&recipientExists); err != nil || !recipientExists {
		return nil
	}

	if !shouldCreateNotificationForType(db, p.RecipientID, p.Type) {
		return nil
	}

	title := strings.TrimSpace(p.Title)
	if utf8.RuneCountInString(title) > 500 {
		runes := []rune(title)
		title = string(runes[:500])
	}
	preview := strings.TrimSpace(p.Preview)
	if utf8.RuneCountInString(preview) > 1000 {
		runes := []rune(preview)
		preview = string(runes[:1000])
	}

	var actorIDArg any
	if p.ActorID != 0 {
		actorIDArg = p.ActorID
	}

	_, err := db.Exec(`
		INSERT INTO notifications (recipient_id, actor_id, type, source_type, source_id, source_public_id, title, preview)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT DO NOTHING`,
		p.RecipientID, actorIDArg, p.Type, p.SourceType, p.SourceID, p.SourcePublicID, title, preview)
	return err
}

// shouldCreateNotificationForType решает, нужно ли создавать уведомление,
// глядя на тогл соответствующей категории в settings юзера.
// Если тип неизвестен или тогл не определён — пропускает (создаёт).
func shouldCreateNotificationForType(db *sql.DB, userID int64, notifType string) bool {
	var col string
	switch notifType {
	case "post_like", "comment_like":
		col = "notif_reactions"
	case "post_comment", "comment_reply", "forum_reply":
		col = "notif_reactions"
	case "mention":
		col = "notif_mentions"
	case "friend_request", "friend_accepted":
		col = "notif_friend_requests"
	case "chat_message":
		col = "notif_chat_messages"
	default:
		return true
	}

	var enabled bool
	err := db.QueryRow(fmt.Sprintf(`SELECT %s FROM users WHERE id = $1 AND is_deleted = FALSE`, col), userID).Scan(&enabled)
	if err != nil {
		return true
	}
	return enabled
}

// getUserDisplayName возвращает отображаемое имя юзера (full_name, или handle если имени нет).
// Используется в текстах уведомлений.
func getUserDisplayName(db *sql.DB, userID int64) string {
	if userID == 0 {
		return "Кто-то"
	}
	var fullName, handle string
	err := db.QueryRow(`SELECT COALESCE(full_name, ''), COALESCE(handle, '') FROM users WHERE id = $1`, userID).Scan(&fullName, &handle)
	if err != nil {
		return "Кто-то"
	}
	fullName = strings.TrimSpace(fullName)
	if fullName != "" {
		return fullName
	}
	if handle != "" {
		return "@" + handle
	}
	return "Кто-то"
}

// truncateRunes обрезает строку до n рун (без обрыва символа Unicode), добавляя «…» если обрезано.
func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	runes := []rune(s)
	if n <= 1 {
		return string(runes[:n])
	}
	return string(runes[:n-1]) + "…"
}

func handleAccountError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, errForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, errNotFound), errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, errConflict), errors.Is(err, errHandleTaken):
		writeError(w, http.StatusConflict, err.Error())
	default:
		log.Printf("account error: %v", err)
		writeError(w, http.StatusInternalServerError, "Ошибка сервера")
	}
}

func validateHandle(h string) (string, error) {
	h = strings.TrimSpace(strings.TrimPrefix(h, "@"))
	lower := strings.ToLower(h)
	if !handleRe.MatchString(lower) {
		return "", fmt.Errorf("%w: handle должен быть 3–24 символа, латиница, цифры или _, начинаться с буквы или цифры", errValidation)
	}
	if strings.Contains(lower, "__") {
		return "", fmt.Errorf("%w: двойные подчёркивания запрещены", errValidation)
	}
	if _, reserved := reservedHandles[lower]; reserved {
		return "", fmt.Errorf("%w: этот handle зарезервирован", errValidation)
	}
	return h, nil
}

func handleFriendActionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, errConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, errNotFound), errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "Ошибка сервера")
	}
}

func listFriends(db *sql.DB, userID int64) ([]friendDTO, error) {
	rows, err := db.Query(`
		SELECT u.public_id, u.full_name, u.email
		FROM friend_requests fr
		JOIN users u ON u.id = CASE
			WHEN fr.requester_id = $1 THEN fr.addressee_id
			ELSE fr.requester_id
		END
		WHERE fr.status = 'accepted'
		  AND ($1 IN (fr.requester_id, fr.addressee_id))
		ORDER BY LOWER(u.full_name)
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []friendDTO
	for rows.Next() {
		var dto friendDTO
		if scanErr := rows.Scan(&dto.FriendID, &dto.FriendName, &dto.Email); scanErr != nil {
			return nil, scanErr
		}
		result = append(result, dto)
	}
	return result, rows.Err()
}

func listIncomingFriendRequests(db *sql.DB, userID int64) ([]friendDTO, error) {
	rows, err := db.Query(`
		SELECT u.public_id, u.full_name, u.email
		FROM friend_requests fr
		JOIN users u ON u.id = fr.requester_id
		WHERE fr.addressee_id = $1
		  AND fr.status = 'pending'
		ORDER BY LOWER(u.full_name)
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []friendDTO
	for rows.Next() {
		var dto friendDTO
		if scanErr := rows.Scan(&dto.FriendID, &dto.FriendName, &dto.Email); scanErr != nil {
			return nil, scanErr
		}
		result = append(result, dto)
	}
	return result, rows.Err()
}

func listOutgoingFriendRequests(db *sql.DB, userID int64) ([]friendDTO, error) {
	rows, err := db.Query(`
		SELECT u.public_id, u.full_name, u.email
		FROM friend_requests fr
		JOIN users u ON u.id = fr.addressee_id
		WHERE fr.requester_id = $1
		  AND fr.status = 'pending'
		ORDER BY LOWER(u.full_name)
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []friendDTO
	for rows.Next() {
		var dto friendDTO
		if scanErr := rows.Scan(&dto.FriendID, &dto.FriendName, &dto.Email); scanErr != nil {
			return nil, scanErr
		}
		result = append(result, dto)
	}
	return result, rows.Err()
}

func listFriendCandidates(db *sql.DB, userID int64, query string) ([]friendCandidateDTO, error) {
	trimmed := strings.TrimSpace(query)
	var (
		rows *sql.Rows
		err  error
	)
	if trimmed == "" {
		rows, err = db.Query(`
		SELECT u.public_id, u.full_name, u.email
		FROM users u
		WHERE u.id <> $1
		  AND NOT EXISTS (
		      SELECT 1
		      FROM friend_requests fr
		      WHERE (
		           (fr.requester_id = $1 AND fr.addressee_id = u.id)
		        OR (fr.requester_id = u.id AND fr.addressee_id = $1)
		      )
		      AND fr.status IN ('pending', 'accepted')
		  )
		ORDER BY LOWER(u.full_name)
		LIMIT 50
	`, userID)
	} else {
		esc := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.ToLower(trimmed))
		search := "%" + esc + "%"
		rows, err = db.Query(`
		SELECT u.public_id, u.full_name, u.email
		FROM users u
		WHERE u.id <> $1
		  AND (
		      LOWER(u.full_name) LIKE $2 ESCAPE '\'
		      OR LOWER(u.email) LIKE $2 ESCAPE '\'
		  )
		  AND NOT EXISTS (
		      SELECT 1
		      FROM friend_requests fr
		      WHERE (
		           (fr.requester_id = $1 AND fr.addressee_id = u.id)
		        OR (fr.requester_id = u.id AND fr.addressee_id = $1)
		      )
		      AND fr.status IN ('pending', 'accepted')
		  )
		ORDER BY LOWER(u.full_name)
		LIMIT 50
	`, userID, search)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []friendCandidateDTO
	for rows.Next() {
		var dto friendCandidateDTO
		if scanErr := rows.Scan(&dto.ID, &dto.FullName, &dto.Email); scanErr != nil {
			return nil, scanErr
		}
		result = append(result, dto)
	}
	return result, rows.Err()
}

func findUserPublicIDByNameOrEmail(db *sql.DB, userID int64, value string) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", fmt.Errorf("%w: укажите имя или email", errValidation)
	}
	vLower := strings.ToLower(v)
	if isValidEmail(vLower) {
		var publicID string
		err := db.QueryRow(`
			SELECT public_id
			FROM users
			WHERE id <> $1
			  AND LOWER(email) = $2
			LIMIT 1
		`, userID, vLower).Scan(&publicID)
		return publicID, err
	}
	rows, err := db.Query(`
		SELECT public_id
		FROM users
		WHERE id <> $1
		  AND LOWER(full_name) = $2
		ORDER BY id
		LIMIT 2
	`, userID, vLower)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if scanErr := rows.Scan(&id); scanErr != nil {
			return "", scanErr
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", sql.ErrNoRows
	}
	if len(ids) > 1 {
		return "", fmt.Errorf("%w: Найдено несколько пользователей с таким именем, используйте email", errValidation)
	}
	return ids[0], nil
}

func createFriendRequest(db *sql.DB, requesterID int64, targetPublicID string) error {
	targetPublicID = strings.TrimSpace(targetPublicID)
	if targetPublicID == "" {
		return fmt.Errorf("%w: id пользователя обязателен", errValidation)
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if rerr := tx.Rollback(); rerr != nil && !errors.Is(rerr, sql.ErrTxDone) {
			log.Printf("tx rollback error: %v", rerr)
		}
	}()

	var addresseeID int64
	if err := tx.QueryRow(`SELECT id FROM users WHERE public_id = $1`, targetPublicID).Scan(&addresseeID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: пользователь не найден", errNotFound)
		}
		return err
	}
	if addresseeID == requesterID {
		return fmt.Errorf("%w: нельзя добавить себя в друзья", errValidation)
	}

	// outcome: что в итоге произошло
	//   "request_sent" — заявка ушла адресату
	//   "auto_accepted" — встречная заявка от адресата автоматически принята
	//   "" — пропускаем уведомление
	outcome := ""
	var notifyTargetID int64

	var existingStatus string
	var existingRequesterID, existingAddresseeID int64
	err = tx.QueryRow(`
		SELECT requester_id, addressee_id, status
		FROM friend_requests
		WHERE (requester_id = $1 AND addressee_id = $2)
		   OR (requester_id = $2 AND addressee_id = $1)
		LIMIT 1
	`, requesterID, addresseeID).Scan(&existingRequesterID, &existingAddresseeID, &existingStatus)
	if err == nil {
		switch existingStatus {
		case "accepted":
			return fmt.Errorf("%w: вы уже друзья", errConflict)
		case "pending":
			if existingRequesterID == requesterID {
				return fmt.Errorf("%w: заявка уже отправлена", errConflict)
			}
			// Встречная заявка от него → принимаем
			if _, err = tx.Exec(`
				UPDATE friend_requests
				SET status = 'accepted', updated_at = NOW()
				WHERE requester_id = $1 AND addressee_id = $2
			`, existingRequesterID, existingAddresseeID); err != nil {
				return err
			}
			// Уведомляем оригинального автора заявки (existingRequesterID == addresseeID),
			// что мы её приняли
			outcome = "auto_accepted"
			notifyTargetID = existingRequesterID
		case "rejected", "canceled", "unfriended":
			if _, err = tx.Exec(`
				UPDATE friend_requests
				SET requester_id = $1, addressee_id = $2, status = 'pending', updated_at = NOW()
				WHERE requester_id = $3 AND addressee_id = $4
			`, requesterID, addresseeID, existingRequesterID, existingAddresseeID); err != nil {
				return err
			}
			outcome = "request_sent"
			notifyTargetID = addresseeID
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	} else {
		// Новая заявка
		if _, err = tx.Exec(`
			INSERT INTO friend_requests(requester_id, addressee_id, status)
			VALUES ($1, $2, 'pending')
		`, requesterID, addresseeID); err != nil {
			return err
		}
		outcome = "request_sent"
		notifyTargetID = addresseeID
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Уведомление — best-effort, после commit
	if outcome != "" && notifyTargetID != 0 && notifyTargetID != requesterID {
		actorName := getUserDisplayName(db, requesterID)
		switch outcome {
		case "request_sent":
			title := actorName + " отправил вам заявку в друзья"
			if err := createNotification(db, createNotificationParams{
				RecipientID: notifyTargetID,
				ActorID:     requesterID,
				Type:        "friend_request",
				SourceType:  "user",
				SourceID:    requesterID,
				Title:       title,
			}); err != nil {
				log.Printf("notif friend_request: %v", err)
			}
		case "auto_accepted":
			title := actorName + " принял вашу заявку в друзья"
			if err := createNotification(db, createNotificationParams{
				RecipientID: notifyTargetID,
				ActorID:     requesterID,
				Type:        "friend_accepted",
				SourceType:  "user",
				SourceID:    requesterID,
				Title:       title,
			}); err != nil {
				log.Printf("notif friend_accepted: %v", err)
			}
		}
	}
	return nil
}
func acceptFriendRequest(db *sql.DB, userID int64, requesterPublicID string) error {
	requesterPublicID = strings.TrimSpace(requesterPublicID)

	// Сначала находим requesterID для уведомления
	var requesterID int64
	if err := db.QueryRow(`SELECT id FROM users WHERE public_id = $1`, requesterPublicID).Scan(&requesterID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: пользователь не найден", errNotFound)
		}
		return err
	}

	res, err := db.Exec(`
		UPDATE friend_requests
		SET status = 'accepted', updated_at = NOW()
		WHERE requester_id = $1
		  AND addressee_id = $2
		  AND status = 'pending'
	`, requesterID, userID)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("%w: заявка не найдена", errNotFound)
	}

	// Уведомляем отправителя что мы приняли его заявку — best-effort
	if requesterID != 0 && requesterID != userID {
		actorName := getUserDisplayName(db, userID)
		title := actorName + " принял вашу заявку в друзья"
		if err := createNotification(db, createNotificationParams{
			RecipientID: requesterID,
			ActorID:     userID,
			Type:        "friend_accepted",
			SourceType:  "user",
			SourceID:    userID,
			Title:       title,
		}); err != nil {
			log.Printf("notif friend_accepted: %v", err)
		}
	}
	return nil
}
func rejectFriendRequest(db *sql.DB, userID int64, requesterPublicID string) error {
	res, err := db.Exec(`
		UPDATE friend_requests fr
		SET status = 'rejected', updated_at = NOW()
		FROM users u
		WHERE fr.requester_id = u.id
		  AND fr.addressee_id = $1
		  AND fr.status = 'pending'
		  AND u.public_id = $2
	`, userID, strings.TrimSpace(requesterPublicID))
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("%w: заявка не найдена", errNotFound)
	}
	return nil
}

func cancelFriendRequest(db *sql.DB, userID int64, addresseePublicID string) error {
	res, err := db.Exec(`
		UPDATE friend_requests fr
		SET status = 'canceled', updated_at = NOW()
		FROM users u
		WHERE fr.addressee_id = u.id
		  AND fr.requester_id = $1
		  AND fr.status = 'pending'
		  AND u.public_id = $2
	`, userID, strings.TrimSpace(addresseePublicID))
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("%w: заявка не найдена", errNotFound)
	}
	return nil
}

func removeFriend(db *sql.DB, userID int64, friendPublicID string) error {
	res, err := db.Exec(`
		UPDATE friend_requests fr
		SET status = 'unfriended', updated_at = NOW()
		FROM users u
		WHERE u.public_id = $2
		  AND fr.status = 'accepted'
		  AND (
		     (fr.requester_id = $1 AND fr.addressee_id = u.id)
		     OR
		     (fr.requester_id = u.id AND fr.addressee_id = $1)
		  )
	`, userID, strings.TrimSpace(friendPublicID))
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("%w: друг не найден", errNotFound)
	}
	return nil
}

type communityListFilters struct {
	Privacy  string
	Category string
	Region   string
	Query    string
}

func listCommunities(db *sql.DB, authUserID int64, hasAuth bool, filters communityListFilters) ([]community, error) {
	var currentUserID sql.NullInt64
	if hasAuth {
		currentUserID = sql.NullInt64{Int64: authUserID, Valid: true}
	}
	args := []any{currentUserID}
	query := `
		SELECT c.id,
		       c.name,
		       c.category,
		       c.description,
		       COALESCE(c.region, ''),
		       COALESCE(c.website, ''),
		       COALESCE(c.email, ''),
		       COALESCE(c.phone, ''),
		       COALESCE(c.avatar_url, ''),
		       COALESCE(c.cover_url, ''),
		       COALESCE(c.color, ''),
		       COALESCE(array_to_json(c.tags), '[]'::json),
		       c.privacy_level,
		       u.public_id,
		       (
		           $1::bigint IS NOT NULL
		           AND EXISTS (
		               SELECT 1
		               FROM community_members cm_user
		               WHERE cm_user.community_id = c.id
		                 AND cm_user.user_id = $1::bigint
		           )
		       ) AS is_member,
		       COALESCE(cm_user.role, ''),
		       COALESCE(m.members_count, 0),
		       c.created_at
		FROM communities c
		JOIN users u ON u.id = c.creator_id
		LEFT JOIN community_members cm_user
		       ON cm_user.community_id = c.id
		      AND cm_user.user_id = $1::bigint
		LEFT JOIN (
			SELECT community_id, COUNT(*)::int AS members_count
			FROM community_members
			GROUP BY community_id
		) m ON m.community_id = c.id
		WHERE c.is_deleted = FALSE`
	if p := normalizeCommunityPrivacy(filters.Privacy); p != "" {
		query += fmt.Sprintf(" AND c.privacy_level = $%d", len(args)+1)
		args = append(args, p)
	}
	if cat := strings.TrimSpace(filters.Category); cat != "" {
		query += fmt.Sprintf(" AND c.category = $%d", len(args)+1)
		args = append(args, cat)
	}
	if region := strings.TrimSpace(filters.Region); region != "" {
		query += fmt.Sprintf(" AND c.region ILIKE $%d", len(args)+1)
		args = append(args, "%"+region+"%")
	}
	if q := strings.TrimSpace(filters.Query); q != "" {
		query += fmt.Sprintf(" AND (c.name ILIKE $%d OR c.description ILIKE $%d)", len(args)+1, len(args)+1)
		args = append(args, "%"+q+"%")
	}
	query += " ORDER BY c.created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []community
	for rows.Next() {
		var item community
		var rawTags []byte
		if scanErr := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Category,
			&item.Description,
			&item.Region,
			&item.Website,
			&item.Email,
			&item.Phone,
			&item.AvatarURL,
			&item.CoverURL,
			&item.Color,
			&rawTags,
			&item.Privacy,
			&item.CreatorID,
			&item.IsMember,
			&item.MyRole,
			&item.Members,
			&item.CreatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		if err := json.Unmarshal(rawTags, &item.Tags); err != nil {
			return nil, fmt.Errorf("decode community tags: %w", err)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func getCommunityByID(db *sql.DB, communityID, authUserID int64, hasAuth bool) (community, error) {
	var item community
	var tagsJSON []byte
	var currentUserID sql.NullInt64
	if hasAuth {
		currentUserID = sql.NullInt64{Int64: authUserID, Valid: true}
	}
	err := db.QueryRow(`
		SELECT c.id,
		       c.name,
		       c.category,
		       COALESCE(c.description, ''),
		       COALESCE(c.region, ''),
		       COALESCE(c.website, ''),
		       COALESCE(c.email, ''),
		       COALESCE(c.phone, ''),
		       COALESCE(c.avatar_url, ''),
		       COALESCE(c.cover_url, ''),
		       COALESCE(c.color, ''),
		       COALESCE(array_to_json(c.tags), '[]'::json),
		       c.privacy_level,
		       u.public_id,
		       (
		           $2::bigint IS NOT NULL
		           AND EXISTS (
		               SELECT 1
		               FROM community_members cm_user
		               WHERE cm_user.community_id = c.id
		                 AND cm_user.user_id = $2::bigint
		           )
		       ) AS is_member,
		       COALESCE(cm_user.role, ''),
		       (SELECT COUNT(*)::int FROM community_members cm WHERE cm.community_id = c.id) AS members_count,
		       c.created_at
		FROM communities c
		LEFT JOIN users u ON u.id = c.creator_id
		LEFT JOIN community_members cm_user
		       ON cm_user.community_id = c.id
		      AND cm_user.user_id = $2::bigint
		WHERE c.id = $1 AND c.is_deleted = FALSE
	`, communityID, currentUserID).Scan(
		&item.ID,
		&item.Name,
		&item.Category,
		&item.Description,
		&item.Region,
		&item.Website,
		&item.Email,
		&item.Phone,
		&item.AvatarURL,
		&item.CoverURL,
		&item.Color,
		&tagsJSON,
		&item.Privacy,
		&item.CreatorID,
		&item.IsMember,
		&item.MyRole,
		&item.Members,
		&item.CreatedAt,
	)
	if err != nil {
		return community{}, err
	}
	if err := json.Unmarshal(tagsJSON, &item.Tags); err != nil {
		return community{}, fmt.Errorf("decode community tags: %w", err)
	}
	if item.Privacy == "closed" && !item.IsMember && !canModerateCommunity(item.MyRole) {
		item.Description = ""
		item.Region = ""
		item.Website = ""
		item.Email = ""
		item.Phone = ""
		item.CoverURL = ""
		item.Tags = []string{}
	}
	return item, nil
}

func createCommunity(db *sql.DB, creatorID int64, req createCommunityRequest) (community, error) {
	name, err := validateCommunityName(req.Name)
	if err != nil {
		return community{}, err
	}
	category := validateCommunityCategory(req.Category)
	privacy := normalizeCommunityPrivacy(req.PrivacyLevel)
	if privacy == "" {
		return community{}, fmt.Errorf("%w: privacy_level должен быть open или closed", errValidation)
	}
	description, err := validateCommunityDescription(req.Description)
	if err != nil {
		return community{}, err
	}
	region, err := validateRegion(req.Region)
	if err != nil {
		return community{}, err
	}
	website, err := validateWebsite(req.Website)
	if err != nil {
		return community{}, err
	}
	email, err := validateEmail(req.Email)
	if err != nil {
		return community{}, err
	}
	phone, err := validatePhone(req.Phone)
	if err != nil {
		return community{}, err
	}
	avatarURL, err := validateAvatarURL(req.AvatarURL)
	if err != nil {
		return community{}, err
	}
	color, err := validateColor(req.Color)
	if err != nil {
		return community{}, err
	}
	tags, err := validateTags(req.Tags)
	if err != nil {
		return community{}, err
	}

	tx, err := db.Begin()
	if err != nil {
		return community{}, err
	}
	defer func() {
		if rerr := tx.Rollback(); rerr != nil && !errors.Is(rerr, sql.ErrTxDone) {
			log.Printf("tx rollback error: %v", rerr)
		}
	}()

	var created community
	var rawTags []byte
	var pgTags pgtype.FlatArray[string] = tags
	err = tx.QueryRow(`
		INSERT INTO communities(name, category, description, region, website, email, phone, avatar_url, color, tags, privacy_level, creator_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::text[], $11, $12)
		RETURNING id, name, category, description, region, website, email, phone, avatar_url, cover_url, color,
			COALESCE(array_to_json(tags), '[]'::json), privacy_level, created_at,
			(SELECT public_id FROM users WHERE id = creator_id)
	`, name, category, description, region, website, email, phone, avatarURL, color, pgTags, privacy, creatorID).
		Scan(&created.ID, &created.Name, &created.Category, &created.Description, &created.Region, &created.Website, &created.Email, &created.Phone, &created.AvatarURL, &created.CoverURL, &created.Color, &rawTags, &created.Privacy, &created.CreatedAt, &created.CreatorID)
	if err != nil {
		return community{}, err
	}
	if err := json.Unmarshal(rawTags, &created.Tags); err != nil {
		return community{}, fmt.Errorf("decode created community tags: %w", err)
	}
	if _, err := tx.Exec(`
		INSERT INTO community_members(community_id, user_id, role)
		VALUES ($1, $2, 'owner')
		ON CONFLICT (community_id, user_id) DO NOTHING
	`, created.ID, creatorID); err != nil {
		return community{}, err
	}
	if err := tx.Commit(); err != nil {
		return community{}, err
	}
	created.Members = 1
	created.IsMember = true
	created.MyRole = "owner"
	return created, nil
}

func joinCommunity(db *sql.DB, communityID, userID int64) error {
	var privacy string
	if err := db.QueryRow(`SELECT privacy_level FROM communities WHERE id = $1 AND is_deleted = FALSE`, communityID).Scan(&privacy); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: сообщество не найдено", errNotFound)
		}
		return err
	}
	if role, err := getUserCommunityRole(db, communityID, userID); err != nil {
		return err
	} else if role != "" {
		return fmt.Errorf("%w: пользователь уже состоит в сообществе", errAlreadyMember)
	}
	if privacy == "closed" {
		return fmt.Errorf("%w: сообщество закрытое, отправьте заявку", errForbidden)
	}
	if _, err := db.Exec(`
		INSERT INTO community_members(community_id, user_id, role)
		VALUES ($1, $2, 'member')
		ON CONFLICT (community_id, user_id) DO NOTHING
	`, communityID, userID); err != nil {
		return err
	}
	return nil
}

func leaveCommunity(db *sql.DB, communityID, userID int64) error {
	role, err := getUserCommunityRole(db, communityID, userID)
	if err != nil {
		return err
	}
	if role == "" {
		return fmt.Errorf("%w: участник не найден", errNotFound)
	}
	if role == "owner" {
		return fmt.Errorf("%w: Сначала передайте роль владельца другому участнику или удалите сообщество", errForbidden)
	}
	result, err := db.Exec(`
		DELETE FROM community_members
		WHERE community_id = $1 AND user_id = $2
	`, communityID, userID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errNotFound
	}
	return nil
}

func normalizePrivacyLevel(value string) string {
	return normalizeCommunityPrivacy(value)
}

func normalizeCommunityPrivacy(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "open":
		return "open"
	case "closed":
		return "closed"
	default:
		return ""
	}
}

func validateCommunityName(s string) (string, error) {
	s = strings.TrimSpace(s)
	if count := utf8.RuneCountInString(s); count < 3 || count > 120 {
		return "", fmt.Errorf("%w: название сообщества должно быть от 3 до 120 символов", errValidation)
	}
	return s, nil
}

func validateCommunityCategory(s string) string {
	s = strings.TrimSpace(s)
	switch s {
	case "Логистика", "ВЭД", "IT", "Регионы", "Карьера", "Другое":
		return s
	default:
		return "Другое"
	}
}

func validateCommunityDescription(s string) (string, error) {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > 1200 {
		return "", fmt.Errorf("%w: описание слишком длинное", errValidation)
	}
	return s, nil
}

func validateRegion(s string) (string, error) {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > 120 {
		return "", fmt.Errorf("%w: регион слишком длинный", errValidation)
	}
	return s, nil
}

func validateWebsite(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if len(s) > 500 || !(strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")) {
		return "", fmt.Errorf("%w: website должен начинаться с http:// или https:// и быть не длиннее 500", errValidation)
	}
	return s, nil
}

func validateEmail(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if _, err := mail.ParseAddress(s); err != nil {
		return "", fmt.Errorf("%w: email некорректен", errValidation)
	}
	return s, nil
}

func validatePhone(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if utf8.RuneCountInString(s) < 5 || utf8.RuneCountInString(s) > 32 {
		return "", fmt.Errorf("%w: телефон должен быть от 5 до 32 символов", errValidation)
	}
	if ok, _ := regexp.MatchString(`^[0-9+\-()\s]+$`, s); !ok {
		return "", fmt.Errorf("%w: телефон содержит недопустимые символы", errValidation)
	}
	return s, nil
}

func validateColor(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if ok, _ := regexp.MatchString(`^#[0-9A-Fa-f]{6}$`, s); !ok {
		return "", fmt.Errorf("%w: color должен быть HEX-цветом вида #RRGGBB", errValidation)
	}
	return s, nil
}

func validateAvatarURL(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if len(s) > 500 || !(strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")) {
		return "", fmt.Errorf("%w: avatar_url должен начинаться с http:// или https:// и быть не длиннее 500", errValidation)
	}
	return s, nil
}

func validateTags(tags []string) ([]string, error) {
	out := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, t := range tags {
		tag := strings.TrimSpace(t)
		if tag == "" {
			continue
		}
		if count := utf8.RuneCountInString(tag); count < 1 || count > 40 {
			return nil, fmt.Errorf("%w: каждый тег должен быть длиной 1..40 символов", errValidation)
		}
		key := strings.ToLower(tag)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, tag)
		if len(out) > 10 {
			return nil, fmt.Errorf("%w: максимум 10 тегов", errValidation)
		}
	}
	return out, nil
}

func canManageCommunity(role string) bool { return role == "owner" || role == "admin" }
func canModerateCommunity(role string) bool {
	return role == "owner" || role == "admin" || role == "moderator"
}

func getUserCommunityRole(db *sql.DB, communityID, userID int64) (string, error) {
	var role string
	err := db.QueryRow(`SELECT role FROM community_members WHERE community_id=$1 AND user_id=$2`, communityID, userID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return role, err
}

func updateCommunity(db *sql.DB, communityID, userID int64, req updateCommunityRequest) (community, error) {
	role, err := getUserCommunityRole(db, communityID, userID)
	if err != nil {
		return community{}, err
	}
	if !canManageCommunity(role) {
		return community{}, fmt.Errorf("%w: недостаточно прав", errForbidden)
	}
	var sets []string
	var args []any
	i := 1
	if req.Name != nil {
		v, err := validateCommunityName(*req.Name)
		if err != nil {
			return community{}, err
		}
		sets = append(sets, fmt.Sprintf("name=$%d", i))
		args = append(args, v)
		i++
	}
	if req.Category != nil {
		sets = append(sets, fmt.Sprintf("category=$%d", i))
		args = append(args, validateCommunityCategory(*req.Category))
		i++
	}
	if req.Description != nil {
		v, err := validateCommunityDescription(*req.Description)
		if err != nil {
			return community{}, err
		}
		sets = append(sets, fmt.Sprintf("description=$%d", i))
		args = append(args, v)
		i++
	}
	if req.Region != nil {
		v, err := validateRegion(*req.Region)
		if err != nil {
			return community{}, err
		}
		sets = append(sets, fmt.Sprintf("region=$%d", i))
		args = append(args, v)
		i++
	}
	if req.Website != nil {
		v, err := validateWebsite(*req.Website)
		if err != nil {
			return community{}, err
		}
		sets = append(sets, fmt.Sprintf("website=$%d", i))
		args = append(args, v)
		i++
	}
	if req.Email != nil {
		v, err := validateEmail(*req.Email)
		if err != nil {
			return community{}, err
		}
		sets = append(sets, fmt.Sprintf("email=$%d", i))
		args = append(args, v)
		i++
	}
	if req.Phone != nil {
		v, err := validatePhone(*req.Phone)
		if err != nil {
			return community{}, err
		}
		sets = append(sets, fmt.Sprintf("phone=$%d", i))
		args = append(args, v)
		i++
	}
	if req.PrivacyLevel != nil {
		v := normalizeCommunityPrivacy(*req.PrivacyLevel)
		if v == "" {
			return community{}, fmt.Errorf("%w: privacy_level должен быть open или closed", errValidation)
		}
		sets = append(sets, fmt.Sprintf("privacy_level=$%d", i))
		args = append(args, v)
		i++
	}
	if req.AvatarURL != nil {
		v, err := validateAvatarURL(*req.AvatarURL)
		if err != nil {
			return community{}, err
		}
		sets = append(sets, fmt.Sprintf("avatar_url=$%d", i))
		args = append(args, v)
		i++
	}
	if req.CoverURL != nil {
		v, err := validateAvatarURL(*req.CoverURL)
		if err != nil {
			return community{}, err
		}
		sets = append(sets, fmt.Sprintf("cover_url=$%d", i))
		args = append(args, v)
		i++
	}
	if req.Color != nil {
		v, err := validateColor(*req.Color)
		if err != nil {
			return community{}, err
		}
		sets = append(sets, fmt.Sprintf("color=$%d", i))
		args = append(args, v)
		i++
	}
	if req.Tags != nil {
		v, err := validateTags(*req.Tags)
		if err != nil {
			return community{}, err
		}
		sets = append(sets, fmt.Sprintf("tags=$%d::text[]", i))
		args = append(args, pgtype.FlatArray[string](v))
		i++
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at=NOW()")
		args = append(args, communityID)
		if _, err := db.Exec("UPDATE communities SET "+strings.Join(sets, ", ")+" WHERE id=$"+strconv.Itoa(i)+" AND is_deleted = FALSE", args...); err != nil {
			return community{}, err
		}
	}
	return getCommunityByID(db, communityID, userID, true)
}

func deleteCommunity(db *sql.DB, communityID, userID int64) error {
	role, err := getUserCommunityRole(db, communityID, userID)
	if err != nil {
		return err
	}
	if role != "owner" {
		return fmt.Errorf("%w: удалить сообщество может только владелец", errForbidden)
	}
	res, err := db.Exec(`UPDATE communities SET is_deleted=TRUE, updated_at=NOW() WHERE id=$1 AND is_deleted=FALSE`, communityID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errNotFound
	}
	return nil
}

func createJoinRequest(db *sql.DB, communityID, userID int64, message string) error {
	message = strings.TrimSpace(message)
	if utf8.RuneCountInString(message) > 500 {
		return fmt.Errorf("%w: сообщение не должно превышать 500 символов", errValidation)
	}
	var privacy string
	if err := db.QueryRow(`SELECT privacy_level FROM communities WHERE id=$1 AND is_deleted=FALSE`, communityID).Scan(&privacy); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errNotFound
		}
		return err
	}
	if privacy != "closed" {
		return fmt.Errorf("%w: заявки доступны только для закрытых сообществ", errValidation)
	}
	if role, err := getUserCommunityRole(db, communityID, userID); err != nil {
		return err
	} else if role != "" {
		return errAlreadyMember
	}
	_, err := db.Exec(`INSERT INTO community_join_requests(community_id, user_id, message) VALUES ($1,$2,$3)`, communityID, userID, message)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return errRequestPending
		}
		return err
	}
	return nil
}

func listJoinRequests(db *sql.DB, communityID, userID int64) ([]communityJoinRequestDTO, error) {
	role, err := getUserCommunityRole(db, communityID, userID)
	if err != nil {
		return nil, err
	}
	if !canModerateCommunity(role) {
		return nil, errForbidden
	}
	rows, err := db.Query(`
		SELECT r.id, u.public_id, u.full_name, r.message, r.status, r.created_at
		FROM community_join_requests r
		JOIN users u ON u.id = r.user_id
		WHERE r.community_id = $1 AND r.status = 'pending'
		ORDER BY r.created_at DESC
	`, communityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []communityJoinRequestDTO
	for rows.Next() {
		var item communityJoinRequestDTO
		if err := rows.Scan(&item.ID, &item.UserPublicID, &item.FullName, &item.Message, &item.Status, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func approveJoinRequest(db *sql.DB, communityID, requestID, deciderID int64) error {
	return decideJoinRequest(db, communityID, requestID, deciderID, "approved")
}

func rejectJoinRequest(db *sql.DB, communityID, requestID, deciderID int64) error {
	return decideJoinRequest(db, communityID, requestID, deciderID, "rejected")
}

func decideJoinRequest(db *sql.DB, communityID, requestID, deciderID int64, status string) error {
	role, err := getUserCommunityRole(db, communityID, deciderID)
	if err != nil {
		return err
	}
	if !canModerateCommunity(role) {
		return errForbidden
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var applicantID int64
	if err := tx.QueryRow(`
		UPDATE community_join_requests
		SET status = $3, decided_at = NOW(), decided_by = $4
		WHERE id = $1 AND community_id = $2 AND status = 'pending'
		RETURNING user_id
	`, requestID, communityID, status, deciderID).Scan(&applicantID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errNotFound
		}
		return err
	}
	if status == "approved" {
		if _, err := tx.Exec(`
			INSERT INTO community_members(community_id, user_id, role)
			VALUES ($1, $2, 'member')
			ON CONFLICT (community_id, user_id) DO NOTHING
		`, communityID, applicantID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func cancelMyJoinRequest(db *sql.DB, communityID, userID int64) error {
	res, err := db.Exec(`
		UPDATE community_join_requests
		SET status='canceled', decided_at=NOW(), decided_by=$3
		WHERE community_id=$1 AND user_id=$2 AND status='pending'
	`, communityID, userID, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errNotFound
	}
	return nil
}

func listCommunityMembers(db *sql.DB, communityID, authUserID int64, hasAuth bool, limit, offset int) ([]communityMemberDTO, error) {
	var privacy string
	if err := db.QueryRow(`SELECT privacy_level FROM communities WHERE id=$1 AND is_deleted=FALSE`, communityID).Scan(&privacy); err != nil {
		return nil, err
	}
	if privacy == "closed" {
		if !hasAuth {
			return nil, errForbidden
		}
		role, err := getUserCommunityRole(db, communityID, authUserID)
		if err != nil {
			return nil, err
		}
		if role == "" {
			return nil, errForbidden
		}
	}
	rows, err := db.Query(`
		SELECT u.public_id, u.full_name, COALESCE(u.avatar_url, ''), cm.role, cm.joined_at
		FROM community_members cm
		JOIN users u ON u.id = cm.user_id
		WHERE cm.community_id = $1
		ORDER BY CASE cm.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 WHEN 'moderator' THEN 2 ELSE 3 END, cm.joined_at ASC
		LIMIT $2 OFFSET $3
	`, communityID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []communityMemberDTO
	for rows.Next() {
		var m communityMemberDTO
		if err := rows.Scan(&m.UserPublicID, &m.FullName, &m.AvatarURL, &m.Role, &m.JoinedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func changeCommunityRole(db *sql.DB, communityID int64, targetUserPublicID, newRole string, actorID int64) error {
	if newRole != "admin" && newRole != "moderator" && newRole != "member" {
		return fmt.Errorf("%w: роль должна быть admin/moderator/member", errValidation)
	}
	actorRole, err := getUserCommunityRole(db, communityID, actorID)
	if err != nil {
		return err
	}
	if actorRole != "owner" {
		return errForbidden
	}
	var targetID int64
	if err := db.QueryRow(`SELECT id FROM users WHERE public_id=$1`, targetUserPublicID).Scan(&targetID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errNotFound
		}
		return err
	}
	if targetID == actorID {
		return fmt.Errorf("%w: нельзя менять свою роль", errValidation)
	}
	var currentRole string
	if err := db.QueryRow(`SELECT role FROM community_members WHERE community_id=$1 AND user_id=$2`, communityID, targetID).Scan(&currentRole); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errNotFound
		}
		return err
	}
	if currentRole == "owner" {
		return fmt.Errorf("%w: нельзя менять роль владельца", errForbidden)
	}
	_, err = db.Exec(`UPDATE community_members SET role=$3 WHERE community_id=$1 AND user_id=$2`, communityID, targetID, newRole)
	return err
}

func kickCommunityMember(db *sql.DB, communityID int64, targetUserPublicID string, actorID int64) error {
	actorRole, err := getUserCommunityRole(db, communityID, actorID)
	if err != nil {
		return err
	}
	if actorRole != "owner" && actorRole != "admin" {
		return errForbidden
	}
	var targetID int64
	var targetRole string
	if err := db.QueryRow(`
		SELECT u.id, cm.role
		FROM users u
		JOIN community_members cm ON cm.user_id = u.id AND cm.community_id = $1
		WHERE u.public_id = $2
	`, communityID, targetUserPublicID).Scan(&targetID, &targetRole); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errNotFound
		}
		return err
	}
	if targetID == actorID {
		return fmt.Errorf("%w: нельзя исключить себя", errValidation)
	}
	if targetRole == "owner" {
		return errForbidden
	}
	if actorRole == "admin" && targetRole == "admin" {
		return errForbidden
	}
	_, err = db.Exec(`DELETE FROM community_members WHERE community_id=$1 AND user_id=$2`, communityID, targetID)
	return err
}

func listCommunityPosts(db *sql.DB, communityID, userID int64, hasAuth bool, limit int, beforeID int64) ([]post, *int64, error) {
	var privacy string
	if err := db.QueryRow(`SELECT privacy_level FROM communities WHERE id=$1 AND is_deleted=FALSE`, communityID).Scan(&privacy); err != nil {
		return nil, nil, err
	}
	if privacy == "closed" {
		if !hasAuth {
			return nil, nil, errForbidden
		}
		role, err := getUserCommunityRole(db, communityID, userID)
		if err != nil {
			return nil, nil, err
		}
		if role == "" {
			return nil, nil, errForbidden
		}
	}
	currentUser := sql.NullInt64{}
	if hasAuth {
		currentUser = sql.NullInt64{Int64: userID, Valid: true}
	}
	args := []any{communityID, limit + 1, currentUser}
	query := `
		SELECT p.id, p.public_id, p.type, p.title, p.content, COALESCE(p.cover_url, ''),
		       COALESCE(array_to_json(p.tags), '[]'::json), p.privacy_level, p.likes_count, p.comments_count,
		       p.views_count, p.saves_count, p.reposts_count, COALESCE(p.reposted_from_id, 0), p.created_at, p.author_id,
		       COALESCE(u.public_id, ''), COALESCE(u.full_name, ''), COALESCE(NULLIF(u.position, ''), u.company_name, ''), COALESCE(u.avatar_url, ''),
		       FALSE,
		       COALESCE(($3::bigint IS NOT NULL AND EXISTS (SELECT 1 FROM post_saves ps WHERE ps.post_id = p.id AND ps.user_id = $3::bigint)), FALSE),
		       COALESCE(($3::bigint IS NOT NULL AND EXISTS (SELECT 1 FROM posts rp WHERE rp.author_id = $3::bigint AND rp.reposted_from_id = p.id AND rp.is_deleted = FALSE)), FALSE),
		       COALESCE(c.name, ''), COALESCE(c.id, 0)
		FROM posts p
		JOIN users u ON u.id = p.author_id
		LEFT JOIN communities c ON c.id = p.community_id
		WHERE p.community_id = $1 AND p.is_deleted = FALSE`
	if beforeID > 0 {
		query += fmt.Sprintf(" AND p.id < $%d", len(args)+1)
		args = append(args, beforeID)
	}
	query += " ORDER BY p.created_at DESC, p.id DESC LIMIT $2"
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var items []post
	for rows.Next() {
		var item post
		var tagsJSON []byte
		if err := rows.Scan(&item.ID, &item.PublicID, &item.Type, &item.Title, &item.Content, &item.CoverURL, &tagsJSON,
			&item.PrivacyLevel, &item.LikesCount, &item.CommentsCount, &item.ViewsCount, &item.SavesCount, &item.RepostsCount, &item.RepostedFromID, &item.CreatedAt, &item.AuthorID,
			&item.AuthorPublicID, &item.AuthorName, &item.AuthorRole, &item.AuthorAvatar, &item.IsLiked, &item.IsSaved, &item.IsReposted, &item.CommunityName, &item.CommunityID); err != nil {
			return nil, nil, err
		}
		_ = json.Unmarshal(tagsJSON, &item.Tags)
		item.Text = item.Content
		if err := populateRepost(db, &item, userID, hasAuth); err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	var next *int64
	if len(items) > limit {
		last := items[limit-1].ID
		next = &last
		items = items[:limit]
	}
	return items, next, rows.Err()
}

func createCommunityPost(db *sql.DB, communityID, authorID int64, req createPostRequest) (post, error) {
	role, err := getUserCommunityRole(db, communityID, authorID)
	if err != nil {
		return post{}, err
	}
	if role == "" {
		return post{}, errForbidden
	}
	title := strings.TrimSpace(req.Title)
	if count := utf8.RuneCountInString(title); count < 3 || count > 200 {
		return post{}, fmt.Errorf("%w: заголовок должен быть от 3 до 200 символов", errValidation)
	}
	content := strings.TrimSpace(req.Content)
	if count := utf8.RuneCountInString(content); count < 10 || count > 20000 {
		return post{}, fmt.Errorf("%w: текст должен быть от 10 до 20000 символов", errValidation)
	}
	tags, err := validateTags(req.Tags)
	if err != nil {
		return post{}, err
	}
	publicID, err := newPublicPostID()
	if err != nil {
		return post{}, err
	}
	var created post
	var tagsJSON []byte
	if err := db.QueryRow(`
		INSERT INTO posts (public_id, author_id, community_id, type, title, content, tags, privacy_level)
		VALUES ($1,$2,$3,'news',$4,$5,$6::text[],'public')
		RETURNING id, public_id, type, title, content, COALESCE(cover_url,''), COALESCE(array_to_json(tags),'[]'::json),
		          privacy_level, likes_count, comments_count, views_count, saves_count, reposts_count, COALESCE(reposted_from_id, 0), created_at, author_id
	`, publicID, authorID, communityID, title, content, pgtype.FlatArray[string](tags)).Scan(
		&created.ID, &created.PublicID, &created.Type, &created.Title, &created.Content, &created.CoverURL, &tagsJSON,
		&created.PrivacyLevel, &created.LikesCount, &created.CommentsCount, &created.ViewsCount, &created.SavesCount, &created.RepostsCount, &created.RepostedFromID, &created.CreatedAt, &created.AuthorID,
	); err != nil {
		return post{}, err
	}
	_ = json.Unmarshal(tagsJSON, &created.Tags)
	created.Text = created.Content
	created.CommunityID = communityID
	_ = saveMentions(db, "post", created.ID, authorID, content, content)
	return hydratePostAuthor(db, created)
}

func deleteCommunityPost(db *sql.DB, communityID int64, postPublicID string, userID int64) error {
	var authorID int64
	if err := db.QueryRow(`SELECT author_id FROM posts WHERE public_id=$1 AND community_id=$2 AND is_deleted=FALSE`, postPublicID, communityID).Scan(&authorID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errNotFound
		}
		return err
	}
	if authorID != userID {
		role, err := getUserCommunityRole(db, communityID, userID)
		if err != nil {
			return err
		}
		if !canModerateCommunity(role) {
			return errForbidden
		}
	}
	_, err := db.Exec(`UPDATE posts SET is_deleted=TRUE, updated_at=NOW() WHERE public_id=$1 AND community_id=$2`, postPublicID, communityID)
	return err
}

func listUserCommunities(db *sql.DB, userPublicID string, authUserID int64, hasAuth bool) ([]community, error) {
	var userID int64
	if err := db.QueryRow(`SELECT id FROM users WHERE public_id=$1`, userPublicID).Scan(&userID); err != nil {
		return nil, err
	}
	var currentUserID sql.NullInt64
	if hasAuth {
		currentUserID = sql.NullInt64{Int64: authUserID, Valid: true}
	}
	rows, err := db.Query(`
		SELECT c.id, c.name, c.category, c.description, COALESCE(c.region,''), COALESCE(c.website,''), COALESCE(c.email,''), COALESCE(c.phone,''),
		       COALESCE(c.avatar_url,''), COALESCE(c.cover_url,''), COALESCE(c.color,''), COALESCE(array_to_json(c.tags), '[]'::json),
		       c.privacy_level, cu.public_id,
		       ($2::bigint IS NOT NULL AND EXISTS (SELECT 1 FROM community_members cmx WHERE cmx.community_id=c.id AND cmx.user_id=$2::bigint)) AS is_member,
		       COALESCE(cm2.role,''), (SELECT COUNT(*)::int FROM community_members cmcnt WHERE cmcnt.community_id=c.id), c.created_at
		FROM communities c
		JOIN community_members cm ON cm.community_id=c.id
		JOIN users cu ON cu.id = c.creator_id
		LEFT JOIN community_members cm2 ON cm2.community_id=c.id AND cm2.user_id=$2::bigint
		WHERE cm.user_id=$1 AND c.is_deleted=FALSE
		ORDER BY cm.joined_at DESC
	`, userID, currentUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []community
	for rows.Next() {
		var item community
		var tagsJSON []byte
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.Description, &item.Region, &item.Website, &item.Email, &item.Phone,
			&item.AvatarURL, &item.CoverURL, &item.Color, &tagsJSON, &item.Privacy, &item.CreatorID, &item.IsMember, &item.MyRole, &item.Members, &item.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(tagsJSON, &item.Tags)
		out = append(out, item)
	}
	return out, rows.Err()
}

func parseCommunityID(s string) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return id, err == nil && id > 0
}

func handleCommunityError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, errForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, errNotFound), errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, errConflict), errors.Is(err, errAlreadyMember), errors.Is(err, errRequestPending):
		writeError(w, http.StatusConflict, err.Error())
	default:
		log.Printf("community error: %v", err)
		writeError(w, http.StatusInternalServerError, "Ошибка сервера")
	}
}

func newPublicPostID() (string, error) {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "p" + hex.EncodeToString(buf), nil
}

func createPost(db *sql.DB, authorID int64, req createPostRequest) (post, error) {
	title := strings.TrimSpace(req.Title)
	if count := utf8.RuneCountInString(title); count < 3 || count > 200 {
		return post{}, fmt.Errorf("%w: заголовок должен быть от 3 до 200 символов", errValidation)
	}
	content := strings.TrimSpace(req.Content)
	if count := utf8.RuneCountInString(content); count < 10 || count > 20000 {
		return post{}, fmt.Errorf("%w: текст должен быть от 10 до 20000 символов", errValidation)
	}
	postType := strings.ToLower(strings.TrimSpace(req.Type))
	if postType == "" {
		postType = "news"
	}
	if postType != "news" && postType != "project" && postType != "note" && postType != "case" {
		return post{}, fmt.Errorf("%w: type должен быть news/project/note/case", errValidation)
	}
	privacy := strings.ToLower(strings.TrimSpace(req.PrivacyLevel))
	if privacy == "" {
		privacy = "public"
	}
	if privacy != "public" && privacy != "friends" && privacy != "private" {
		return post{}, fmt.Errorf("%w: privacy_level должен быть public/friends/private", errValidation)
	}
	coverURL := strings.TrimSpace(req.CoverURL)
	if coverURL != "" {
		if utf8.RuneCountInString(coverURL) > 500 || (!strings.HasPrefix(coverURL, "http://") && !strings.HasPrefix(coverURL, "https://")) {
			return post{}, fmt.Errorf("%w: cover_url должен начинаться с http:// или https:// и быть не длиннее 500 символов", errValidation)
		}
	}
	tags := make([]string, 0, len(req.Tags))
	seen := make(map[string]struct{}, len(req.Tags))
	for _, raw := range req.Tags {
		tag := strings.TrimSpace(raw)
		if tag == "" {
			continue
		}
		if utf8.RuneCountInString(tag) > 40 {
			return post{}, fmt.Errorf("%w: тег должен быть не длиннее 40 символов", errValidation)
		}
		key := strings.ToLower(tag)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		tags = append(tags, tag)
		if len(tags) > 10 {
			return post{}, fmt.Errorf("%w: максимум 10 тегов", errValidation)
		}
	}

	var pgTags pgtype.FlatArray[string] = tags
	for attempts := 0; attempts < 5; attempts++ {
		publicID, err := newPublicPostID()
		if err != nil {
			return post{}, err
		}
		created, err := scanPost(db.QueryRow(`
			INSERT INTO posts (public_id, author_id, type, title, content, cover_url, tags, privacy_level)
			VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7::text[], $8)
			RETURNING id, public_id, type, title, content, COALESCE(cover_url, ''), COALESCE(array_to_json(tags), '[]'::json),
					  privacy_level, likes_count, comments_count, views_count, saves_count, reposts_count, COALESCE(reposted_from_id, 0), created_at, author_id
		`, publicID, authorID, postType, title, content, coverURL, pgTags, privacy))
		if err == nil {
			_ = saveMentions(db, "post", created.ID, authorID, content, content)
			return hydratePostAuthor(db, created)
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			continue
		}
		return post{}, err
	}
	return post{}, fmt.Errorf("failed to allocate post public_id")
}

func getPostByID(db *sql.DB, publicID string, authUserID int64, hasAuth, incrementViews bool) (post, error) {
	return getPostByIDInternal(db, publicID, authUserID, hasAuth, incrementViews, true)
}

func getPostByIDInternal(db *sql.DB, publicID string, authUserID int64, hasAuth, incrementViews, includeRepost bool) (post, error) {
	_ = incrementViews
	tx, err := db.Begin()
	if err != nil {
		return post{}, err
	}
	defer tx.Rollback()
	currentUser := sql.NullInt64{}
	if hasAuth {
		currentUser = sql.NullInt64{Int64: authUserID, Valid: true}
	}
	var item post
	var tagsJSON []byte
	var coverURL string
	err = tx.QueryRow(`
		SELECT p.id, p.public_id, p.type, p.title, p.content, COALESCE(p.cover_url, ''),
		       COALESCE(array_to_json(p.tags), '[]'::json), p.privacy_level, p.likes_count,
		       p.comments_count, p.views_count, p.created_at, p.author_id,
		       COALESCE(pl.user_id IS NOT NULL, FALSE),
		       p.saves_count, p.reposts_count, COALESCE(p.reposted_from_id, 0),
		       COALESCE(ps.user_id IS NOT NULL, FALSE),
		       COALESCE(($2::bigint IS NOT NULL AND EXISTS (
		           SELECT 1 FROM posts rp WHERE rp.author_id = $2::bigint AND rp.reposted_from_id = p.id AND rp.is_deleted = FALSE
		       )), FALSE)
		FROM posts p
		LEFT JOIN post_likes pl ON pl.post_id = p.id AND pl.user_id = $2::bigint
		LEFT JOIN post_saves ps ON ps.post_id = p.id AND ps.user_id = $2::bigint
		WHERE p.public_id = $1 AND p.is_deleted = FALSE
		LIMIT 1
	`, publicID, currentUser).Scan(
		&item.ID, &item.PublicID, &item.Type, &item.Title, &item.Content, &coverURL, &tagsJSON,
		&item.PrivacyLevel, &item.LikesCount, &item.CommentsCount, &item.ViewsCount, &item.CreatedAt, &item.AuthorID, &item.IsLiked,
		&item.SavesCount, &item.RepostsCount, &item.RepostedFromID, &item.IsSaved, &item.IsReposted,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return post{}, errNotFound
		}
		return post{}, err
	}
	item.CoverURL = coverURL
	item.Text = item.Content
	_ = json.Unmarshal(tagsJSON, &item.Tags)
	if item.PrivacyLevel == "private" && (!hasAuth || authUserID != item.AuthorID) {
		return post{}, errNotFound
	}
	if item.PrivacyLevel == "friends" && (!hasAuth || !canViewFriendsPost(tx, authUserID, item.AuthorID)) {
		return post{}, errNotFound
	}
	if err := tx.Commit(); err != nil {
		return post{}, err
	}
	item, err = hydratePostAuthor(db, item)
	if err != nil {
		return post{}, err
	}
	if includeRepost {
		if err := populateRepost(db, &item, authUserID, hasAuth); err != nil {
			return post{}, err
		}
	}
	return item, nil
}

func listFeed(db *sql.DB, authUserID int64, hasAuth bool, limit int, beforeID int64, postType string) ([]post, *int64, error) {
	currentUser := sql.NullInt64{}
	if hasAuth {
		currentUser = sql.NullInt64{Int64: authUserID, Valid: true}
	}
	args := []any{currentUser, limit + 1}
	query := `
		SELECT p.id, p.public_id, p.type, p.title, p.content, COALESCE(p.cover_url, ''),
		       COALESCE(array_to_json(p.tags), '[]'::json), p.privacy_level, p.likes_count, p.comments_count,
		       p.views_count, p.saves_count, p.reposts_count, COALESCE(p.reposted_from_id, 0), p.created_at, p.author_id,
		       COALESCE(u.public_id, ''), COALESCE(u.full_name, ''), COALESCE(NULLIF(u.position, ''), u.company_name, ''), COALESCE(u.avatar_url, ''),
		       COALESCE(pl.user_id IS NOT NULL, FALSE),
		       COALESCE(ps.user_id IS NOT NULL, FALSE),
		       COALESCE(($1::bigint IS NOT NULL AND EXISTS (SELECT 1 FROM posts rp WHERE rp.author_id = $1::bigint AND rp.reposted_from_id = p.id AND rp.is_deleted = FALSE)), FALSE),
		       COALESCE(c.name, ''), COALESCE(c.id, 0)
		FROM posts p
		JOIN users u ON u.id = p.author_id
		LEFT JOIN post_likes pl ON pl.post_id = p.id AND pl.user_id = $1::bigint
		LEFT JOIN post_saves ps ON ps.post_id = p.id AND ps.user_id = $1::bigint
		LEFT JOIN communities c ON c.id = p.community_id
		WHERE p.is_deleted = FALSE AND p.privacy_level = 'public'`
	if postType != "" {
		postType = strings.ToLower(strings.TrimSpace(postType))
		query += fmt.Sprintf(" AND p.type = $%d", len(args)+1)
		args = append(args, postType)
	}
	if beforeID > 0 {
		query += fmt.Sprintf(" AND p.id < $%d", len(args)+1)
		args = append(args, beforeID)
	}
	query += " ORDER BY p.created_at DESC, p.id DESC LIMIT $2"
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var items []post
	for rows.Next() {
		var item post
		var tagsJSON []byte
		if err := rows.Scan(&item.ID, &item.PublicID, &item.Type, &item.Title, &item.Content, &item.CoverURL, &tagsJSON,
			&item.PrivacyLevel, &item.LikesCount, &item.CommentsCount, &item.ViewsCount, &item.SavesCount, &item.RepostsCount, &item.RepostedFromID, &item.CreatedAt, &item.AuthorID,
			&item.AuthorPublicID, &item.AuthorName, &item.AuthorRole, &item.AuthorAvatar, &item.IsLiked, &item.IsSaved, &item.IsReposted, &item.CommunityName, &item.CommunityID); err != nil {
			return nil, nil, err
		}
		_ = json.Unmarshal(tagsJSON, &item.Tags)
		item.Text = item.Content
		if err := populateRepost(db, &item, authUserID, hasAuth); err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	var nextCursor *int64
	if len(items) > limit {
		last := items[limit-1].ID
		nextCursor = &last
		items = items[:limit]
	}
	return items, nextCursor, nil
}

// listNotifications возвращает список уведомлений юзера с информацией об акторе.
// Параметры: limit, before_id (пагинация назад), only_unread.
func listNotifications(db *sql.DB, userID int64, limit int, beforeID int64, onlyUnread bool) ([]notification, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}

	where := []string{"n.recipient_id = $1"}
	args := []any{userID}
	argIdx := 2

	if beforeID > 0 {
		where = append(where, fmt.Sprintf("n.id < $%d", argIdx))
		args = append(args, beforeID)
		argIdx++
	}
	if onlyUnread {
		where = append(where, "n.is_read = FALSE")
	}

	query := fmt.Sprintf(`
		SELECT n.id, n.type, n.source_type, n.source_id, n.source_public_id,
		       n.title, n.preview, n.is_read, n.created_at,
		       COALESCE(u.public_id, ''), COALESCE(u.full_name, ''), COALESCE(u.avatar_url, '')
		FROM notifications n
		LEFT JOIN users u ON u.id = n.actor_id
		WHERE %s
		ORDER BY n.created_at DESC, n.id DESC
		LIMIT %d`, strings.Join(where, " AND "), limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []notification
	for rows.Next() {
		var n notification
		if err := rows.Scan(&n.ID, &n.Type, &n.SourceType, &n.SourceID, &n.SourcePublicID,
			&n.Title, &n.Preview, &n.IsRead, &n.CreatedAt,
			&n.ActorPublicID, &n.ActorName, &n.ActorAvatar); err != nil {
			return nil, err
		}
		if n.ActorName != "" {
			n.ActorColor = stableColorForName(n.ActorName)
		}
		items = append(items, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if items == nil {
		items = []notification{}
	}
	return items, nil
}

func countUnreadNotifications(db *sql.DB, userID int64) (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE recipient_id = $1 AND is_read = FALSE`, userID).Scan(&n)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func markNotificationRead(db *sql.DB, userID int64, notifID int64) error {
	res, err := db.Exec(`UPDATE notifications SET is_read = TRUE WHERE id = $1 AND recipient_id = $2 AND is_read = FALSE`, notifID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil
	}
	return nil
}

func markAllNotificationsRead(db *sql.DB, userID int64) error {
	_, err := db.Exec(`UPDATE notifications SET is_read = TRUE WHERE recipient_id = $1 AND is_read = FALSE`, userID)
	return err
}

// generateProjectPublicID генерирует уникальный public_id для проекта формата "p_<base36>".
func generateProjectPublicID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "p_" + hex.EncodeToString(b)
}

// validateProjectStatus проверяет что статус — один из допустимых.
func validateProjectStatus(s string) bool {
	switch s {
	case "planned", "active", "paused", "done":
		return true
	}
	return false
}

// listProjects возвращает список проектов с фильтрами и сортировкой.
func listProjects(db *sql.DB, viewerID int64, filters map[string]string) ([]project, error) {
	where := []string{"p.is_deleted = FALSE"}
	args := []any{viewerID}
	argIdx := 2

	if cat := strings.TrimSpace(filters["category"]); cat != "" {
		where = append(where, fmt.Sprintf("p.category = $%d", argIdx))
		args = append(args, cat)
		argIdx++
	}
	if st := strings.TrimSpace(filters["status"]); st != "" && validateProjectStatus(st) {
		where = append(where, fmt.Sprintf("p.status = $%d", argIdx))
		args = append(args, st)
		argIdx++
	}
	if filters["owner_only"] == "1" && viewerID > 0 {
		where = append(where, fmt.Sprintf("p.owner_id = $%d", argIdx))
		args = append(args, viewerID)
		argIdx++
	}
	if q := strings.TrimSpace(filters["search"]); q != "" {
		where = append(where, fmt.Sprintf("(p.title ILIKE $%d OR p.description ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+q+"%")
		argIdx++
	}

	orderBy := "p.created_at DESC"
	switch filters["sort"] {
	case "deadline":
		orderBy = "p.deadline ASC NULLS LAST, p.created_at DESC"
	case "members":
		orderBy = "members_count DESC, p.created_at DESC"
	}

	limit := 50
	if l := filters["limit"]; l != "" {
		if n, err := strconv.Atoi(l); err == nil && n >= 1 && n <= 100 {
			limit = n
		}
	}

	query := fmt.Sprintf(`
		SELECT p.id, p.public_id, p.owner_id, p.title, p.description, p.category, p.status,
		       p.deadline, p.budget, p.cover_color, COALESCE(array_to_json(p.tags), '[]'::json),
		       p.created_at, p.updated_at,
		       COALESCE(u.public_id, ''), COALESCE(u.full_name, ''), COALESCE(u.avatar_url, ''),
		       (SELECT COUNT(*) FROM project_members WHERE project_id = p.id) AS members_count,
		       EXISTS(SELECT 1 FROM project_members WHERE project_id = p.id AND user_id = $1) AS is_member
		FROM projects p
		LEFT JOIN users u ON u.id = p.owner_id
		WHERE %s
		ORDER BY %s
		LIMIT %d`, strings.Join(where, " AND "), orderBy, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []project
	for rows.Next() {
		var p project
		var tagsJSON []byte
		var membersCount int
		var isMember bool
		if err := rows.Scan(&p.ID, &p.PublicID, &p.OwnerID, &p.Title, &p.Description, &p.Category, &p.Status,
			&p.Deadline, &p.Budget, &p.CoverColor, &tagsJSON,
			&p.CreatedAt, &p.UpdatedAt,
			&p.OwnerPublicID, &p.OwnerName, &p.OwnerAvatar,
			&membersCount, &isMember); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(tagsJSON, &p.Tags)
		if p.Tags == nil {
			p.Tags = []string{}
		}
		p.MembersCount = membersCount
		p.IsMember = isMember
		p.IsOwner = p.OwnerID == viewerID
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if items == nil {
		items = []project{}
	}
	return items, nil
}

// getProject возвращает один проект по public_id, с проверкой is_deleted.
func getProject(db *sql.DB, viewerID int64, publicID string) (project, error) {
	var p project
	var tagsJSON []byte
	err := db.QueryRow(`
		SELECT p.id, p.public_id, p.owner_id, p.title, p.description, p.category, p.status,
		       p.deadline, p.budget, p.cover_color, COALESCE(array_to_json(p.tags), '[]'::json),
		       p.created_at, p.updated_at,
		       COALESCE(u.public_id, ''), COALESCE(u.full_name, ''), COALESCE(u.avatar_url, ''),
		       (SELECT COUNT(*) FROM project_members WHERE project_id = p.id) AS members_count,
		       EXISTS(SELECT 1 FROM project_members WHERE project_id = p.id AND user_id = $1) AS is_member
		FROM projects p
		LEFT JOIN users u ON u.id = p.owner_id
		WHERE p.public_id = $2 AND p.is_deleted = FALSE`, viewerID, publicID).Scan(
		&p.ID, &p.PublicID, &p.OwnerID, &p.Title, &p.Description, &p.Category, &p.Status,
		&p.Deadline, &p.Budget, &p.CoverColor, &tagsJSON,
		&p.CreatedAt, &p.UpdatedAt,
		&p.OwnerPublicID, &p.OwnerName, &p.OwnerAvatar,
		&p.MembersCount, &p.IsMember,
	)
	if err != nil {
		return p, err
	}
	_ = json.Unmarshal(tagsJSON, &p.Tags)
	if p.Tags == nil {
		p.Tags = []string{}
	}
	p.IsOwner = p.OwnerID == viewerID
	return p, nil
}

// listProjectMembers возвращает участников проекта с инфо о юзере.
func listProjectMembers(db *sql.DB, projectID int64) ([]projectMember, error) {
	rows, err := db.Query(`
		SELECT u.public_id, u.full_name, COALESCE(u.avatar_url, ''), pm.role, pm.joined_at
		FROM project_members pm
		JOIN users u ON u.id = pm.user_id
		WHERE pm.project_id = $1
		ORDER BY 
			CASE pm.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END,
			pm.joined_at ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []projectMember
	for rows.Next() {
		var m projectMember
		if err := rows.Scan(&m.UserPublicID, &m.UserName, &m.UserAvatar, &m.Role, &m.JoinedAt); err != nil {
			return nil, err
		}
		m.UserColor = stableColorForName(m.UserName)
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if members == nil {
		members = []projectMember{}
	}
	return members, nil
}

// createProject создаёт проект и автоматически добавляет owner'а в project_members.
func createProject(db *sql.DB, ownerID int64, req createProjectRequest) (project, error) {
	title := strings.TrimSpace(req.Title)
	if utf8.RuneCountInString(title) < 1 || utf8.RuneCountInString(title) > 200 {
		return project{}, fmt.Errorf("%w: длина названия 1..200", errValidation)
	}
	description := strings.TrimSpace(req.Description)
	if utf8.RuneCountInString(description) > 5000 {
		return project{}, fmt.Errorf("%w: описание до 5000 символов", errValidation)
	}
	status := req.Status
	if status == "" {
		status = "active"
	}
	if !validateProjectStatus(status) {
		return project{}, fmt.Errorf("%w: некорректный статус", errValidation)
	}
	coverColor := strings.TrimSpace(req.CoverColor)
	if coverColor == "" {
		coverColor = "#1E8A4C"
	}
	if !regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`).MatchString(coverColor) {
		coverColor = "#1E8A4C"
	}
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}
	if len(tags) > 10 {
		tags = tags[:10]
	}

	publicID := generateProjectPublicID()
	var pid int64
	err := db.QueryRow(`
		INSERT INTO projects (public_id, owner_id, title, description, category, status, deadline, budget, cover_color, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`,
		publicID, ownerID, title, description, strings.TrimSpace(req.Category), status,
		req.Deadline, req.Budget, coverColor, tags,
	).Scan(&pid)
	if err != nil {
		return project{}, err
	}

	_, err = db.Exec(`
		INSERT INTO project_members (project_id, user_id, role)
		VALUES ($1, $2, 'owner')
		ON CONFLICT DO NOTHING`, pid, ownerID)
	if err != nil {
		return project{}, err
	}

	return getProject(db, ownerID, publicID)
}

// updateProject — только owner или admin. Возвращает обновлённый проект.
func updateProject(db *sql.DB, viewerID int64, publicID string, req updateProjectRequest) (project, error) {
	var ownerID int64
	var role sql.NullString
	err := db.QueryRow(`
		SELECT p.owner_id, pm.role
		FROM projects p
		LEFT JOIN project_members pm ON pm.project_id = p.id AND pm.user_id = $1
		WHERE p.public_id = $2 AND p.is_deleted = FALSE`, viewerID, publicID).Scan(&ownerID, &role)
	if err != nil {
		return project{}, err
	}
	if ownerID != viewerID && (!role.Valid || (role.String != "owner" && role.String != "admin")) {
		return project{}, fmt.Errorf("%w: только владелец или администратор может редактировать", errForbidden)
	}

	sets := []string{}
	args := []any{}
	argIdx := 1

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if utf8.RuneCountInString(title) < 1 || utf8.RuneCountInString(title) > 200 {
			return project{}, fmt.Errorf("%w: длина названия 1..200", errValidation)
		}
		sets = append(sets, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, title)
		argIdx++
	}
	if req.Description != nil {
		desc := strings.TrimSpace(*req.Description)
		if utf8.RuneCountInString(desc) > 5000 {
			return project{}, fmt.Errorf("%w: описание до 5000", errValidation)
		}
		sets = append(sets, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, desc)
		argIdx++
	}
	if req.Category != nil {
		sets = append(sets, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, strings.TrimSpace(*req.Category))
		argIdx++
	}
	if req.Status != nil {
		if !validateProjectStatus(*req.Status) {
			return project{}, fmt.Errorf("%w: некорректный статус", errValidation)
		}
		sets = append(sets, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}
	if req.Deadline != nil {
		sets = append(sets, fmt.Sprintf("deadline = $%d", argIdx))
		args = append(args, *req.Deadline)
		argIdx++
	}
	if req.Budget != nil {
		sets = append(sets, fmt.Sprintf("budget = $%d", argIdx))
		args = append(args, *req.Budget)
		argIdx++
	}
	if req.CoverColor != nil {
		cc := strings.TrimSpace(*req.CoverColor)
		if !regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`).MatchString(cc) {
			cc = "#1E8A4C"
		}
		sets = append(sets, fmt.Sprintf("cover_color = $%d", argIdx))
		args = append(args, cc)
		argIdx++
	}
	if req.Tags != nil {
		tags := *req.Tags
		if len(tags) > 10 {
			tags = tags[:10]
		}
		sets = append(sets, fmt.Sprintf("tags = $%d", argIdx))
		args = append(args, tags)
		argIdx++
	}

	if len(sets) == 0 {
		return getProject(db, viewerID, publicID)
	}

	sets = append(sets, "updated_at = NOW()")
	query := fmt.Sprintf("UPDATE projects SET %s WHERE public_id = $%d", strings.Join(sets, ", "), argIdx)
	args = append(args, publicID)

	if _, err := db.Exec(query, args...); err != nil {
		return project{}, err
	}

	return getProject(db, viewerID, publicID)
}

// deleteProject — soft-delete. Только owner.
func deleteProject(db *sql.DB, viewerID int64, publicID string) error {
	res, err := db.Exec(`
		UPDATE projects SET is_deleted = TRUE, updated_at = NOW()
		WHERE public_id = $1 AND owner_id = $2 AND is_deleted = FALSE`, publicID, viewerID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: проект не найден или нет прав", errForbidden)
	}
	return nil
}

// addProjectMember добавляет участника. Только owner/admin.
func addProjectMember(db *sql.DB, viewerID int64, publicID string, req addProjectMemberRequest) error {
	var pid int64
	var ownerID int64
	var role sql.NullString
	err := db.QueryRow(`
		SELECT p.id, p.owner_id, pm.role
		FROM projects p
		LEFT JOIN project_members pm ON pm.project_id = p.id AND pm.user_id = $1
		WHERE p.public_id = $2 AND p.is_deleted = FALSE`, viewerID, publicID).Scan(&pid, &ownerID, &role)
	if err != nil {
		return err
	}
	if ownerID != viewerID && (!role.Valid || (role.String != "owner" && role.String != "admin")) {
		return fmt.Errorf("%w: только владелец или администратор может добавлять участников", errForbidden)
	}

	var newUserID int64
	if err := db.QueryRow(`SELECT id FROM users WHERE public_id = $1 AND is_deleted = FALSE`, req.UserPublicID).Scan(&newUserID); err != nil {
		return err
	}

	newRole := strings.ToLower(strings.TrimSpace(req.Role))
	if newRole != "admin" && newRole != "member" {
		newRole = "member"
	}

	_, err = db.Exec(`
		INSERT INTO project_members (project_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (project_id, user_id) DO UPDATE SET role = EXCLUDED.role`,
		pid, newUserID, newRole)
	if err != nil {
		return err
	}

	if newUserID != 0 && newUserID != viewerID {
		actorName := getUserDisplayName(db, viewerID)
		var projectTitle, projectPublicID string
		_ = db.QueryRow(`SELECT title, public_id FROM projects WHERE id = $1`, pid).Scan(&projectTitle, &projectPublicID)
		title := actorName + " добавил вас в проект"
		if projectTitle != "" {
			title = actorName + " добавил вас в проект «" + truncateRunes(projectTitle, 80) + "»"
		}
		if err := createNotification(db, createNotificationParams{
			RecipientID:    newUserID,
			ActorID:        viewerID,
			Type:           "project_member_added",
			SourceType:     "project",
			SourceID:       pid,
			SourcePublicID: projectPublicID,
			Title:          title,
		}); err != nil {
			log.Printf("notif project_member_added: %v", err)
		}
	}

	return nil
}

// removeProjectMember удаляет участника. Owner/admin может убрать любого, остальные — только себя.
func removeProjectMember(db *sql.DB, viewerID int64, publicID, targetUserPublicID string) error {
	var pid, ownerID, targetID int64
	var role sql.NullString
	err := db.QueryRow(`
		SELECT p.id, p.owner_id, pm.role
		FROM projects p
		LEFT JOIN project_members pm ON pm.project_id = p.id AND pm.user_id = $1
		WHERE p.public_id = $2 AND p.is_deleted = FALSE`, viewerID, publicID).Scan(&pid, &ownerID, &role)
	if err != nil {
		return err
	}
	if err := db.QueryRow(`SELECT id FROM users WHERE public_id = $1`, targetUserPublicID).Scan(&targetID); err != nil {
		return err
	}
	if targetID == ownerID {
		return fmt.Errorf("%w: владелец не может покинуть свой проект", errForbidden)
	}
	isPriv := ownerID == viewerID || (role.Valid && role.String == "admin")
	if !isPriv && targetID != viewerID {
		return fmt.Errorf("%w: можно удалить только себя", errForbidden)
	}
	_, err = db.Exec(`DELETE FROM project_members WHERE project_id = $1 AND user_id = $2`, pid, targetID)
	return err
}

// listTopPosts возвращает посты, отсортированные по weighted score
// за указанный период. Используется для hero-блока ("Главная новость дня",
// period=day) и блока "Топ недели" (period=week) на /dashboard.html.
//
// score = likes_count*3 + comments_count*2 + saves_count*5 + views_count*0.1
//
// period: "day" (24 часа) или "week" (7 дней). По умолчанию week.
// limit: 1..10. По умолчанию 5.
func listTopPosts(db *sql.DB, authUserID int64, hasAuth bool, period string, limit int) ([]post, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}
	interval := "7 days"
	if period == "day" {
		interval = "24 hours"
	}

	currentUser := sql.NullInt64{}
	if hasAuth {
		currentUser = sql.NullInt64{Int64: authUserID, Valid: true}
	}

	query := fmt.Sprintf(`
		SELECT p.id, p.public_id, p.type, p.title, p.content, COALESCE(p.cover_url, ''),
		       COALESCE(array_to_json(p.tags), '[]'::json), p.privacy_level, p.likes_count, p.comments_count,
		       p.views_count, p.saves_count, p.reposts_count, COALESCE(p.reposted_from_id, 0), p.created_at, p.author_id,
		       COALESCE(u.public_id, ''), COALESCE(u.full_name, ''), COALESCE(NULLIF(u.position, ''), u.company_name, ''), COALESCE(u.avatar_url, ''),
		       COALESCE(pl.user_id IS NOT NULL, FALSE),
		       COALESCE(ps.user_id IS NOT NULL, FALSE),
		       COALESCE(($1::bigint IS NOT NULL AND EXISTS (SELECT 1 FROM posts rp WHERE rp.author_id = $1::bigint AND rp.reposted_from_id = p.id AND rp.is_deleted = FALSE)), FALSE),
		       COALESCE(c.name, ''), COALESCE(c.id, 0)
		FROM posts p
		JOIN users u ON u.id = p.author_id
		LEFT JOIN post_likes pl ON pl.post_id = p.id AND pl.user_id = $1::bigint
		LEFT JOIN post_saves ps ON ps.post_id = p.id AND ps.user_id = $1::bigint
		LEFT JOIN communities c ON c.id = p.community_id
		WHERE p.is_deleted = FALSE
		  AND p.privacy_level = 'public'
		  AND p.type = 'news'
		  AND p.created_at > NOW() - INTERVAL '%s'
		ORDER BY (p.likes_count * 3 + p.comments_count * 2 + p.saves_count * 5 + p.views_count * 0.1) DESC,
		         p.created_at DESC
		LIMIT $2`, interval)

	rows, err := db.Query(query, currentUser, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []post
	for rows.Next() {
		var item post
		var tagsJSON []byte
		if err := rows.Scan(&item.ID, &item.PublicID, &item.Type, &item.Title, &item.Content, &item.CoverURL, &tagsJSON,
			&item.PrivacyLevel, &item.LikesCount, &item.CommentsCount, &item.ViewsCount, &item.SavesCount, &item.RepostsCount, &item.RepostedFromID, &item.CreatedAt, &item.AuthorID,
			&item.AuthorPublicID, &item.AuthorName, &item.AuthorRole, &item.AuthorAvatar, &item.IsLiked, &item.IsSaved, &item.IsReposted, &item.CommunityName, &item.CommunityID); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(tagsJSON, &item.Tags)
		item.Text = item.Content
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func listUserPosts(db *sql.DB, userPublicID string, authUserID int64, hasAuth bool, limit int, beforeID int64) ([]post, *int64, error) {
	var targetID int64
	if err := db.QueryRow(`SELECT id FROM users WHERE public_id = $1`, userPublicID).Scan(&targetID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, errNotFound
		}
		return nil, nil, err
	}
	isOwner := hasAuth && authUserID == targetID
	isFriend := hasAuth && areFriends(db, authUserID, targetID)
	currentUser := sql.NullInt64{}
	if hasAuth {
		currentUser = sql.NullInt64{Int64: authUserID, Valid: true}
	}
	query := `
		SELECT p.id, p.public_id, p.type, p.title, p.content, COALESCE(p.cover_url, ''),
		       COALESCE(array_to_json(p.tags), '[]'::json), p.privacy_level, p.likes_count, p.comments_count,
		       p.views_count, p.saves_count, p.reposts_count, COALESCE(p.reposted_from_id, 0), p.created_at, p.author_id,
		       COALESCE(u.public_id, ''), COALESCE(u.full_name, ''), COALESCE(NULLIF(u.position, ''), u.company_name, ''), COALESCE(u.avatar_url, ''),
		       COALESCE(pl.user_id IS NOT NULL, FALSE),
		       COALESCE(ps.user_id IS NOT NULL, FALSE),
		       COALESCE(($1::bigint IS NOT NULL AND EXISTS (SELECT 1 FROM posts rp WHERE rp.author_id = $1::bigint AND rp.reposted_from_id = p.id AND rp.is_deleted = FALSE)), FALSE),
		       COALESCE(c.name, ''), COALESCE(c.id, 0)
		FROM posts p
		JOIN users u ON u.id = p.author_id
		LEFT JOIN post_likes pl ON pl.post_id = p.id AND pl.user_id = $1::bigint
		LEFT JOIN post_saves ps ON ps.post_id = p.id AND ps.user_id = $1::bigint
		LEFT JOIN communities c ON c.id = p.community_id
		WHERE p.author_id = $2 AND p.is_deleted = FALSE`
	args := []any{currentUser, targetID, limit + 1}
	if !isOwner {
		if isFriend {
			query += " AND p.privacy_level IN ('public', 'friends')"
		} else {
			query += " AND p.privacy_level = 'public'"
		}
	}
	if beforeID > 0 {
		query += fmt.Sprintf(" AND p.id < $%d", len(args)+1)
		args = append(args, beforeID)
	}
	query += " ORDER BY p.created_at DESC, p.id DESC LIMIT $3"
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var items []post
	for rows.Next() {
		var item post
		var tagsJSON []byte
		if err := rows.Scan(&item.ID, &item.PublicID, &item.Type, &item.Title, &item.Content, &item.CoverURL, &tagsJSON,
			&item.PrivacyLevel, &item.LikesCount, &item.CommentsCount, &item.ViewsCount, &item.SavesCount, &item.RepostsCount, &item.RepostedFromID, &item.CreatedAt, &item.AuthorID,
			&item.AuthorPublicID, &item.AuthorName, &item.AuthorRole, &item.AuthorAvatar, &item.IsLiked, &item.IsSaved, &item.IsReposted, &item.CommunityName, &item.CommunityID); err != nil {
			return nil, nil, err
		}
		_ = json.Unmarshal(tagsJSON, &item.Tags)
		item.Text = item.Content
		if err := populateRepost(db, &item, authUserID, hasAuth); err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	var nextCursor *int64
	if len(items) > limit {
		last := items[limit-1].ID
		nextCursor = &last
		items = items[:limit]
	}
	return items, nextCursor, nil
}

func softDeletePost(db *sql.DB, publicID string, userID int64) error {
	res, err := db.Exec(`UPDATE posts SET is_deleted = TRUE, updated_at = NOW() WHERE public_id = $1 AND author_id = $2 AND is_deleted = FALSE`, publicID, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errNotFound
	}
	return nil
}

func togglePostSave(db *sql.DB, postPublicID string, userID int64) (bool, int, error) {
	tx, err := db.Begin()
	if err != nil {
		return false, 0, err
	}
	defer tx.Rollback()

	var postID int64
	if err := tx.QueryRow(`SELECT id FROM posts WHERE public_id = $1 AND is_deleted = FALSE`, postPublicID).Scan(&postID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, 0, errNotFound
		}
		return false, 0, err
	}

	res, err := tx.Exec(`DELETE FROM post_saves WHERE post_id = $1 AND user_id = $2`, postID, userID)
	if err != nil {
		return false, 0, err
	}
	deleted, _ := res.RowsAffected()
	isSaved := deleted == 0
	if isSaved {
		if _, err := tx.Exec(`INSERT INTO post_saves (post_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, postID, userID); err != nil {
			return false, 0, err
		}
	}

	delta := 1
	if !isSaved {
		delta = -1
	}
	if _, err := tx.Exec(`UPDATE posts SET saves_count = GREATEST(0, saves_count + $1), updated_at = NOW() WHERE id = $2`, delta, postID); err != nil {
		return false, 0, err
	}
	var savesCount int
	if err := tx.QueryRow(`SELECT saves_count FROM posts WHERE id = $1`, postID).Scan(&savesCount); err != nil {
		return false, 0, err
	}
	if err := tx.Commit(); err != nil {
		return false, 0, err
	}
	return isSaved, savesCount, nil
}

func createRepost(db *sql.DB, originalPublicID string, userID int64, comment string) (post, error) {
	tx, err := db.Begin()
	if err != nil {
		return post{}, err
	}
	defer tx.Rollback()

	var origID int64
	var origCoverURL string
	var origTags pgtype.FlatArray[string]
	if err := tx.QueryRow(`
		SELECT id, COALESCE(cover_url, ''), COALESCE(tags, '{}'::text[])
		FROM posts WHERE public_id = $1 AND is_deleted = FALSE
	`, originalPublicID).Scan(&origID, &origCoverURL, &origTags); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return post{}, errNotFound
		}
		return post{}, err
	}

	newPublicID, err := newPublicPostID()
	if err != nil {
		return post{}, err
	}

	var insertedID int64
	if err := tx.QueryRow(`
		INSERT INTO posts (public_id, author_id, type, title, content, tags, cover_url, privacy_level, reposted_from_id, created_at, updated_at)
		VALUES ($1, $2, 'news', '', $3, $4::text[], NULLIF($5, ''), 'public', $6, NOW(), NOW())
		RETURNING id
	`, newPublicID, userID, comment, origTags, origCoverURL, origID).Scan(&insertedID); err != nil {
		return post{}, err
	}
	_ = insertedID

	if _, err := tx.Exec(`UPDATE posts SET reposts_count = reposts_count + 1, updated_at = NOW() WHERE id = $1`, origID); err != nil {
		return post{}, err
	}
	if err := tx.Commit(); err != nil {
		return post{}, err
	}
	return getPostByID(db, newPublicID, userID, true, false)
}

func registerPostView(db *sql.DB, postPublicID string, userID int64, ipHash string) (bool, int, error) {
	tx, err := db.Begin()
	if err != nil {
		return false, 0, err
	}
	defer tx.Rollback()
	var postID int64
	if err := tx.QueryRow(`SELECT id FROM posts WHERE public_id = $1 AND is_deleted = FALSE`, postPublicID).Scan(&postID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, 0, errNotFound
		}
		return false, 0, err
	}
	var exists bool
	if userID > 0 {
		err = tx.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM post_views
				WHERE post_id = $1 AND user_id = $2 AND viewed_at > NOW() - INTERVAL '12 hours'
			)
		`, postID, userID).Scan(&exists)
	} else {
		err = tx.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM post_views
				WHERE post_id = $1 AND user_id IS NULL AND ip_hash = $2 AND viewed_at > NOW() - INTERVAL '12 hours'
			)
		`, postID, ipHash).Scan(&exists)
	}
	if err != nil {
		return false, 0, err
	}
	counted := false
	if !exists {
		var userIDArg any
		if userID > 0 {
			userIDArg = userID
		}
		if _, err := tx.Exec(`INSERT INTO post_views (post_id, user_id, ip_hash) VALUES ($1, $2, $3)`, postID, userIDArg, ipHash); err != nil {
			return false, 0, err
		}
		if _, err := tx.Exec(`UPDATE posts SET views_count = views_count + 1, updated_at = NOW() WHERE id = $1`, postID); err != nil {
			return false, 0, err
		}
		counted = true
	}
	var viewsCount int
	if err := tx.QueryRow(`SELECT views_count FROM posts WHERE id = $1`, postID).Scan(&viewsCount); err != nil {
		return false, 0, err
	}
	if err := tx.Commit(); err != nil {
		return false, 0, err
	}
	return counted, viewsCount, nil
}

func listSavedPosts(db *sql.DB, userID int64, limit int) ([]post, error) {
	rows, err := db.Query(`
		SELECT p.public_id
		FROM post_saves ps
		JOIN posts p ON p.id = ps.post_id
		WHERE ps.user_id = $1 AND p.is_deleted = FALSE
		ORDER BY ps.created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	items := make([]post, 0, len(ids))
	for _, id := range ids {
		item, err := getPostByID(db, id, userID, true, false)
		if err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func toggleLike(db *sql.DB, publicID string, userID int64) (bool, int, error) {
	tx, err := db.Begin()
	if err != nil {
		return false, 0, err
	}
	defer tx.Rollback()
	var postID int64
	if err := tx.QueryRow(`SELECT id FROM posts WHERE public_id = $1 AND is_deleted = FALSE`, publicID).Scan(&postID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, 0, errNotFound
		}
		return false, 0, err
	}
	var exists bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM post_likes WHERE post_id = $1 AND user_id = $2)`, postID, userID).Scan(&exists); err != nil {
		return false, 0, err
	}
	isLiked := false
	if exists {
		if _, err := tx.Exec(`DELETE FROM post_likes WHERE post_id = $1 AND user_id = $2`, postID, userID); err != nil {
			return false, 0, err
		}
		if _, err := tx.Exec(`UPDATE posts SET likes_count = likes_count - 1, updated_at = NOW() WHERE id = $1 AND likes_count > 0`, postID); err != nil {
			return false, 0, err
		}
	} else {
		if _, err := tx.Exec(`INSERT INTO post_likes(post_id, user_id) VALUES ($1, $2)`, postID, userID); err != nil {
			return false, 0, err
		}
		if _, err := tx.Exec(`UPDATE posts SET likes_count = likes_count + 1, updated_at = NOW() WHERE id = $1`, postID); err != nil {
			return false, 0, err
		}
		isLiked = true
	}

	var postAuthorID int64
	var postTitle, postPreview string
	if isLiked {
		_ = tx.QueryRow(`SELECT author_id, COALESCE(title, ''), COALESCE(LEFT(content, 200), '') FROM posts WHERE id = $1`, postID).
			Scan(&postAuthorID, &postTitle, &postPreview)
	}

	var likesCount int
	if err := tx.QueryRow(`SELECT likes_count FROM posts WHERE id = $1`, postID).Scan(&likesCount); err != nil {
		return false, 0, err
	}
	if err := tx.Commit(); err != nil {
		return false, 0, err
	}

	if isLiked && postAuthorID != 0 && postAuthorID != userID {
		actorName := getUserDisplayName(db, userID)
		title := actorName + " оценил вашу публикацию"
		if postTitle != "" {
			title = actorName + " оценил «" + truncateRunes(postTitle, 80) + "»"
		}
		if err := createNotification(db, createNotificationParams{
			RecipientID:    postAuthorID,
			ActorID:        userID,
			Type:           "post_like",
			SourceType:     "post",
			SourceID:       postID,
			SourcePublicID: publicID,
			Title:          title,
			Preview:        postPreview,
		}); err != nil {
			log.Printf("notif post_like: %v", err)
		}
	}

	return isLiked, likesCount, nil
}

func listComments(db *sql.DB, postPublicID string, limit int) ([]postComment, error) {
	rows, err := db.Query(`
		SELECT pc.id, u.public_id, u.full_name, pc.content, pc.parent_id, pc.created_at
		FROM post_comments pc
		JOIN posts p ON p.id = pc.post_id
		JOIN users u ON u.id = pc.author_id
		WHERE p.public_id = $1 AND p.is_deleted = FALSE AND pc.is_deleted = FALSE
		ORDER BY pc.created_at ASC, pc.id ASC
		LIMIT $2
	`, postPublicID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []postComment
	for rows.Next() {
		var c postComment
		if err := rows.Scan(&c.ID, &c.AuthorPublicID, &c.AuthorName, &c.Content, &c.ParentID, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func createComment(db *sql.DB, postPublicID string, authorID int64, req createCommentRequest) (postComment, error) {
	content := strings.TrimSpace(req.Content)
	if count := utf8.RuneCountInString(content); count < 1 || count > 5000 {
		return postComment{}, fmt.Errorf("%w: комментарий должен быть от 1 до 5000 символов", errValidation)
	}
	tx, err := db.Begin()
	if err != nil {
		return postComment{}, err
	}
	defer tx.Rollback()
	var postID int64
	if err := tx.QueryRow(`SELECT id FROM posts WHERE public_id = $1 AND is_deleted = FALSE`, postPublicID).Scan(&postID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return postComment{}, errNotFound
		}
		return postComment{}, err
	}
	if req.ParentID != nil {
		var parentPostID int64
		if err := tx.QueryRow(`SELECT post_id FROM post_comments WHERE id = $1 AND is_deleted = FALSE`, *req.ParentID).Scan(&parentPostID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return postComment{}, fmt.Errorf("%w: родительский комментарий не найден", errValidation)
			}
			return postComment{}, err
		}
		if parentPostID != postID {
			return postComment{}, fmt.Errorf("%w: parent_id должен принадлежать тому же посту", errValidation)
		}
	}
	var created postComment
	if err := tx.QueryRow(`
		INSERT INTO post_comments(post_id, author_id, parent_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, parent_id, content, created_at
	`, postID, authorID, req.ParentID, content).Scan(&created.ID, &created.ParentID, &created.Content, &created.CreatedAt); err != nil {
		return postComment{}, err
	}
	if _, err := tx.Exec(`UPDATE posts SET comments_count = comments_count + 1, updated_at = NOW() WHERE id = $1`, postID); err != nil {
		return postComment{}, err
	}
	if err := tx.QueryRow(`SELECT public_id, full_name FROM users WHERE id = $1`, authorID).Scan(&created.AuthorPublicID, &created.AuthorName); err != nil {
		return postComment{}, err
	}
	if err := tx.Commit(); err != nil {
		return postComment{}, err
	}

	go notifyOnComment(db, postID, postPublicID, created.ID, created.ParentID, authorID, content)

	_ = saveMentions(db, "comment", created.ID, authorID, content, content)
	return created, nil
}

// notifyOnComment создаёт уведомление при комментарии:
// - корневой коммент → автору поста ('post_comment')
// - ответ на чужой коммент → автору родительского ('comment_reply')
// Запускать в горутине после tx.Commit (не должна блокировать ответ).
func notifyOnComment(db *sql.DB, postID int64, postPublicID string, commentID int64, parentID *int64, actorID int64, content string) {
	actorName := getUserDisplayName(db, actorID)
	preview := truncateRunes(content, 200)

	if parentID != nil && *parentID > 0 {
		var parentAuthorID int64
		if err := db.QueryRow(`SELECT author_id FROM post_comments WHERE id = $1 AND is_deleted = FALSE`, *parentID).Scan(&parentAuthorID); err == nil && parentAuthorID != 0 && parentAuthorID != actorID {
			title := actorName + " ответил на ваш комментарий"
			if err := createNotification(db, createNotificationParams{
				RecipientID:    parentAuthorID,
				ActorID:        actorID,
				Type:           "comment_reply",
				SourceType:     "comment",
				SourceID:       commentID,
				SourcePublicID: postPublicID,
				Title:          title,
				Preview:        preview,
			}); err != nil {
				log.Printf("notif comment_reply: %v", err)
			}
		}

		var postAuthorID int64
		if err := db.QueryRow(`SELECT author_id FROM posts WHERE id = $1`, postID).Scan(&postAuthorID); err == nil && postAuthorID != 0 && postAuthorID != actorID && postAuthorID != parentAuthorID {
			title := actorName + " прокомментировал вашу публикацию"
			if err := createNotification(db, createNotificationParams{
				RecipientID:    postAuthorID,
				ActorID:        actorID,
				Type:           "post_comment",
				SourceType:     "post",
				SourceID:       postID,
				SourcePublicID: postPublicID,
				Title:          title,
				Preview:        preview,
			}); err != nil {
				log.Printf("notif post_comment (reply): %v", err)
			}
		}
		return
	}

	var postAuthorID int64
	if err := db.QueryRow(`SELECT author_id FROM posts WHERE id = $1`, postID).Scan(&postAuthorID); err == nil && postAuthorID != 0 && postAuthorID != actorID {
		title := actorName + " прокомментировал вашу публикацию"
		if err := createNotification(db, createNotificationParams{
			RecipientID:    postAuthorID,
			ActorID:        actorID,
			Type:           "post_comment",
			SourceType:     "post",
			SourceID:       postID,
			SourcePublicID: postPublicID,
			Title:          title,
			Preview:        preview,
		}); err != nil {
			log.Printf("notif post_comment: %v", err)
		}
	}
}

func scanPost(row *sql.Row) (post, error) {
	var item post
	var tagsJSON []byte
	var coverURL string
	if err := row.Scan(&item.ID, &item.PublicID, &item.Type, &item.Title, &item.Content, &coverURL, &tagsJSON,
		&item.PrivacyLevel, &item.LikesCount, &item.CommentsCount, &item.ViewsCount, &item.SavesCount, &item.RepostsCount, &item.RepostedFromID, &item.CreatedAt, &item.AuthorID); err != nil {
		return post{}, err
	}
	item.CoverURL = coverURL
	item.Text = item.Content
	_ = json.Unmarshal(tagsJSON, &item.Tags)
	return item, nil
}

func hydratePostAuthor(db *sql.DB, item post) (post, error) {
	if err := db.QueryRow(`
		SELECT public_id, full_name, COALESCE(NULLIF(position, ''), company_name, ''), COALESCE(avatar_url, '')
		FROM users WHERE id = $1
	`, item.AuthorID).Scan(&item.AuthorPublicID, &item.AuthorName, &item.AuthorRole, &item.AuthorAvatar); err != nil {
		return post{}, err
	}
	return item, nil
}

func populateRepost(db *sql.DB, item *post, authUserID int64, hasAuth bool) error {
	if item.RepostedFromID <= 0 {
		return nil
	}
	var originalPublicID string
	if err := db.QueryRow(`SELECT public_id FROM posts WHERE id = $1 AND is_deleted = FALSE`, item.RepostedFromID).Scan(&originalPublicID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			item.Repost = &post{Content: "Оригинал удалён", Text: "Оригинал удалён"}
			return nil
		}
		return err
	}
	original, err := getPostByIDInternal(db, originalPublicID, authUserID, hasAuth, false, false)
	if err != nil {
		item.Repost = &post{Content: "Оригинал удалён", Text: "Оригинал удалён"}
		return nil
	}
	original.Repost = nil
	original.RepostedFromID = 0
	item.Repost = &original
	return nil
}

func hashIP(ip string) string {
	h := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(h[:])
}

func canViewFriendsPost(tx *sql.Tx, authUserID, authorID int64) bool {
	if authUserID == authorID {
		return true
	}
	var ok bool
	_ = tx.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM friend_requests
			WHERE status = 'accepted'
			  AND ((requester_id = $1 AND addressee_id = $2) OR (requester_id = $2 AND addressee_id = $1))
		)
	`, authUserID, authorID).Scan(&ok)
	return ok
}

func areFriends(db *sql.DB, a, b int64) bool {
	var ok bool
	_ = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM friend_requests
			WHERE status = 'accepted'
			  AND ((requester_id = $1 AND addressee_id = $2) OR (requester_id = $2 AND addressee_id = $1))
		)
	`, a, b).Scan(&ok)
	return ok
}

func parseLimit(raw string, def, max int) int {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || v <= 0 {
		return def
	}
	if v > max {
		return max
	}
	return v
}

func parseIDOrZero(raw string) int64 {
	v, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || v < 0 {
		return 0
	}
	return v
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func searchEscapeILike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

func handleGlobalSearch(w http.ResponseWriter, r *http.Request, db *sql.DB, sessions *sessionStore) {
	started := time.Now()
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if utf8.RuneCountInString(q) < 2 {
		writeError(w, http.StatusBadRequest, "запрос слишком короткий")
		return
	}
	if utf8.RuneCountInString(q) > 200 {
		q = string([]rune(q)[:200])
	}

	typ := strings.TrimSpace(r.URL.Query().Get("type"))
	if typ == "" {
		typ = "all"
	}
	switch typ {
	case "all", "users", "posts", "communities", "events", "companies", "sections":
	default:
		typ = "all"
	}

	sortBy := strings.TrimSpace(r.URL.Query().Get("sort"))
	if sortBy == "" {
		sortBy = "relevance"
	}
	switch sortBy {
	case "relevance", "recent", "popular":
	default:
		sortBy = "relevance"
	}

	limit := parseIntDefault(strings.TrimSpace(r.URL.Query().Get("limit")), 50)
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}
	offset := parseIntDefault(strings.TrimSpace(r.URL.Query().Get("offset")), 0)
	if offset < 0 {
		offset = 0
	}

	viewerID, _ := optionalAuthenticatedUserID(r, sessions)

	res := searchResult{
		Query: q,
		Counts: map[string]int{
			"users": 0, "posts": 0, "communities": 0, "events": 0, "companies": 0, "sections": 0,
		},
		Users:       []searchUser{},
		Posts:       []searchPost{},
		Communities: []searchCommunity{},
		Events:      []searchEvent{},
		Companies:   []searchCompany{},
		Sections:    []searchSection{},
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	runOne := func(category string, fn func() (int, error)) {
		if typ != "all" && typ != category {
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			n, err := fn()
			if err != nil {
				log.Printf("search %s: %v", category, err)
				return
			}
			mu.Lock()
			res.Counts[category] = n
			mu.Unlock()
		}()
	}

	runOne("users", func() (int, error) { return searchUsersInto(db, q, sortBy, limit, offset, &res, &mu) })
	runOne("posts", func() (int, error) { return searchPostsInto(db, q, sortBy, limit, offset, &res, &mu) })
	runOne("communities", func() (int, error) { return searchCommunitiesInto(db, q, sortBy, limit, offset, viewerID, &res, &mu) })
	runOne("events", func() (int, error) { return searchEventsInto(db, q, sortBy, limit, offset, viewerID, &res, &mu) })
	runOne("companies", func() (int, error) { return searchCompaniesInto(db, q, sortBy, limit, offset, &res, &mu) })

	wg.Wait()

	res.Total = res.Counts["users"] + res.Counts["posts"] + res.Counts["communities"] + res.Counts["events"] + res.Counts["companies"]
	res.TookMS = time.Since(started).Milliseconds()
	writeJSON(w, http.StatusOK, res)
}

func searchUsersInto(db *sql.DB, q, sortBy string, limit, offset int, res *searchResult, mu *sync.Mutex) (int, error) {
	orderBy := "score DESC, full_name"
	if sortBy == "recent" {
		orderBy = "created_at DESC, full_name"
	}
	escaped := searchEscapeILike(q)
	rows, err := db.Query(`
		SELECT
			public_id, full_name, COALESCE(handle,'') AS handle,
			COALESCE(position,'') AS position,
			COALESCE(company_name,'') AS company_name,
			COALESCE(bio,'') AS bio,
			COALESCE(city,'') AS city,
			(
				CASE WHEN LOWER(full_name) = LOWER($1) THEN 10 ELSE 0 END +
				CASE WHEN LOWER(full_name) LIKE LOWER($1) || '%' THEN 5 ELSE 0 END +
				CASE WHEN LOWER(COALESCE(handle,'')) = LOWER($1) THEN 8 ELSE 0 END +
				CASE WHEN LOWER(COALESCE(handle,'')) LIKE LOWER($1) || '%' THEN 4 ELSE 0 END +
				CASE WHEN COALESCE(position,'') ILIKE '%' || $2 || '%' ESCAPE '\' THEN 3 ELSE 0 END +
				CASE WHEN COALESCE(company_name,'') ILIKE '%' || $2 || '%' ESCAPE '\' THEN 3 ELSE 0 END +
				CASE WHEN COALESCE(bio,'') ILIKE '%' || $2 || '%' ESCAPE '\' THEN 1 ELSE 0 END +
				CASE WHEN COALESCE(city,'') ILIKE '%' || $2 || '%' ESCAPE '\' THEN 1 ELSE 0 END
			) AS score
		FROM users
		WHERE is_deleted = FALSE
		  AND (
			full_name ILIKE '%' || $2 || '%' ESCAPE '\' OR
			COALESCE(handle,'') ILIKE '%' || $2 || '%' ESCAPE '\' OR
			COALESCE(position,'') ILIKE '%' || $2 || '%' ESCAPE '\' OR
			COALESCE(company_name,'') ILIKE '%' || $2 || '%' ESCAPE '\' OR
			COALESCE(bio,'') ILIKE '%' || $2 || '%' ESCAPE '\' OR
			COALESCE(city,'') ILIKE '%' || $2 || '%' ESCAPE '\'
		  )
		ORDER BY `+orderBy+`
		LIMIT $3 OFFSET $4
	`, q, escaped, limit, offset)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	out := make([]searchUser, 0, limit)
	for rows.Next() {
		var item searchUser
		if err := rows.Scan(&item.PublicID, &item.FullName, &item.Handle, &item.Position, &item.CompanyName, &item.Bio, &item.City, &item.Score); err != nil {
			return 0, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	mu.Lock()
	res.Users = out
	mu.Unlock()
	return len(out), nil
}

func searchPostsInto(db *sql.DB, q, sortBy string, limit, offset int, res *searchResult, mu *sync.Mutex) (int, error) {
	orderBy := "score DESC, p.created_at DESC"
	if sortBy == "recent" {
		orderBy = "p.created_at DESC"
	} else if sortBy == "popular" {
		orderBy = "COALESCE(p.likes_count, 0) DESC, p.created_at DESC"
	}
	escaped := searchEscapeILike(q)
	rows, err := db.Query(`
		SELECT
			p.public_id,
			LEFT(p.content, 400) AS content,
			'' AS category,
			u.public_id AS author_public_id,
			u.full_name AS author_name,
			COALESCE(p.likes_count, 0) AS likes_count,
			COALESCE(p.comments_count, 0) AS comments_count,
			p.created_at,
			(
				CASE WHEN p.content ILIKE '%' || $1 || '%' ESCAPE '\' THEN 5 ELSE 0 END +
				LN(GREATEST(COALESCE(p.likes_count,0), 1) + 1)
			) AS score
		FROM posts p
		JOIN users u ON u.id = p.author_id
		WHERE COALESCE(p.is_deleted, FALSE) = FALSE
		  AND p.privacy_level = 'public'
		  AND p.content ILIKE '%' || $1 || '%' ESCAPE '\'
		ORDER BY `+orderBy+`
		LIMIT $2 OFFSET $3
	`, escaped, limit, offset)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	out := make([]searchPost, 0, limit)
	for rows.Next() {
		var item searchPost
		if err := rows.Scan(&item.PublicID, &item.Content, &item.Category, &item.AuthorPublicID, &item.AuthorName, &item.LikesCount, &item.CommentsCount, &item.CreatedAt, &item.Score); err != nil {
			return 0, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	mu.Lock()
	res.Posts = out
	mu.Unlock()
	return len(out), nil
}

func searchCommunitiesInto(db *sql.DB, q, sortBy string, limit, offset int, viewerID int64, res *searchResult, mu *sync.Mutex) (int, error) {
	orderBy := "score DESC, name"
	if sortBy == "recent" {
		orderBy = "created_at DESC, name"
	} else if sortBy == "popular" {
		orderBy = "members_count DESC, name"
	}
	escaped := searchEscapeILike(q)
	rows, err := db.Query(`
		SELECT
			public_id, name,
			COALESCE(description,'') AS description,
			COALESCE(region,'') AS region,
			COALESCE(category,'') AS category,
			COALESCE(color,'') AS color,
			COALESCE((SELECT COUNT(*)::int FROM community_members cm WHERE cm.community_id = communities.id), 0) AS members_count,
			COALESCE(privacy_level,'open') AS privacy,
			(
				CASE WHEN LOWER(name) = LOWER($2) THEN 10 ELSE 0 END +
				CASE WHEN LOWER(name) LIKE LOWER($2) || '%' THEN 5 ELSE 0 END +
				CASE WHEN COALESCE(category,'') ILIKE '%' || $1 || '%' ESCAPE '\' THEN 3 ELSE 0 END +
				CASE WHEN COALESCE(region,'') ILIKE '%' || $1 || '%' ESCAPE '\' THEN 2 ELSE 0 END +
				CASE WHEN COALESCE(description,'') ILIKE '%' || $1 || '%' ESCAPE '\' THEN 1 ELSE 0 END
			) AS score
		FROM communities
		WHERE COALESCE(is_deleted, FALSE) = FALSE
		  AND (
			privacy_level = 'open' OR privacy_level IS NULL OR $5 IN (
				SELECT user_id FROM community_members WHERE community_id = communities.id
			)
		  )
		  AND (
			name ILIKE '%' || $1 || '%' ESCAPE '\' OR
			COALESCE(description,'') ILIKE '%' || $1 || '%' ESCAPE '\' OR
			COALESCE(region,'') ILIKE '%' || $1 || '%' ESCAPE '\' OR
			COALESCE(category,'') ILIKE '%' || $1 || '%' ESCAPE '\'
		  )
		ORDER BY `+orderBy+`
		LIMIT $3 OFFSET $4
	`, escaped, q, limit, offset, viewerID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	out := make([]searchCommunity, 0, limit)
	for rows.Next() {
		var item searchCommunity
		if err := rows.Scan(&item.PublicID, &item.Name, &item.Description, &item.Region, &item.Category, &item.Color, &item.MembersCount, &item.Privacy, &item.Score); err != nil {
			return 0, err
		}
		if item.Privacy == "open" {
			item.Privacy = "public"
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	mu.Lock()
	res.Communities = out
	mu.Unlock()
	return len(out), nil
}

func searchEventsInto(db *sql.DB, q, sortBy string, limit, offset int, viewerID int64, res *searchResult, mu *sync.Mutex) (int, error) {
	orderBy := "score DESC, e.starts_at ASC"
	if sortBy == "recent" {
		orderBy = "e.created_at DESC"
	} else if sortBy == "popular" {
		orderBy = "COALESCE(e.registered_count, 0) DESC, e.starts_at ASC"
	}
	escaped := searchEscapeILike(q)
	rows, err := db.Query(`
		SELECT
			e.public_id, e.title,
			LEFT(COALESCE(e.description,''), 400) AS description,
			e.type, e.format,
			COALESCE(e.city,'') AS city,
			e.starts_at,
			COALESCE(e.registered_count, 0) AS registered_count,
			COALESCE(e.banner_color,'') AS banner_color,
			(
				CASE WHEN LOWER(e.title) = LOWER($2) THEN 10 ELSE 0 END +
				CASE WHEN LOWER(e.title) LIKE LOWER($2) || '%' THEN 5 ELSE 0 END +
				CASE WHEN COALESCE(e.category,'') ILIKE '%' || $1 || '%' ESCAPE '\' THEN 3 ELSE 0 END +
				CASE WHEN COALESCE(e.city,'') ILIKE '%' || $1 || '%' ESCAPE '\' THEN 2 ELSE 0 END +
				CASE WHEN COALESCE(e.description,'') ILIKE '%' || $1 || '%' ESCAPE '\' THEN 1 ELSE 0 END +
				LN(GREATEST(COALESCE(e.registered_count,0), 1) + 1)
			) AS score
		FROM events e
		LEFT JOIN communities c ON c.id = e.community_id
		WHERE COALESCE(e.is_deleted, FALSE) = FALSE
		  AND e.status = 'published'
		  AND (
			c.id IS NULL OR c.privacy_level = 'open' OR $5 IN (
				SELECT user_id FROM community_members WHERE community_id = c.id
			)
		  )
		  AND (
			e.title ILIKE '%' || $1 || '%' ESCAPE '\' OR
			COALESCE(e.description,'') ILIKE '%' || $1 || '%' ESCAPE '\' OR
			COALESCE(e.city,'') ILIKE '%' || $1 || '%' ESCAPE '\' OR
			COALESCE(e.category,'') ILIKE '%' || $1 || '%' ESCAPE '\' OR
			EXISTS (SELECT 1 FROM unnest(COALESCE(e.tags,'{}'::text[])) t WHERE t ILIKE '%' || $1 || '%' ESCAPE '\')
		  )
		ORDER BY `+orderBy+`
		LIMIT $3 OFFSET $4
	`, escaped, q, limit, offset, viewerID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	out := make([]searchEvent, 0, limit)
	for rows.Next() {
		var item searchEvent
		if err := rows.Scan(&item.PublicID, &item.Title, &item.Description, &item.Type, &item.Format, &item.City, &item.StartsAt, &item.RegisteredCount, &item.BannerColor, &item.Score); err != nil {
			return 0, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	mu.Lock()
	res.Events = out
	mu.Unlock()
	return len(out), nil
}

func searchCompaniesInto(db *sql.DB, q, sortBy string, limit, offset int, res *searchResult, mu *sync.Mutex) (int, error) {
	orderBy := "score DESC, employee_count DESC"
	if sortBy == "recent" || sortBy == "popular" {
		orderBy = "employee_count DESC, name"
	}
	escaped := searchEscapeILike(q)
	rows, err := db.Query(`
		SELECT
			company_name AS name,
			COUNT(*) AS employee_count,
			MAX(COALESCE(position,'')) AS industry,
			MAX(COALESCE(city,'')) AS city,
			(
				CASE WHEN LOWER(company_name) = LOWER($2) THEN 10 ELSE 0 END +
				CASE WHEN LOWER(company_name) LIKE LOWER($2) || '%' THEN 5 ELSE 0 END
			) AS score
		FROM users
		WHERE is_deleted = FALSE
		  AND COALESCE(company_name,'') <> ''
		  AND company_name ILIKE '%' || $1 || '%' ESCAPE '\'
		GROUP BY company_name
		ORDER BY `+orderBy+`
		LIMIT $3 OFFSET $4
	`, escaped, q, limit, offset)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	out := make([]searchCompany, 0, limit)
	for rows.Next() {
		var item searchCompany
		var employeeCount int
		if err := rows.Scan(&item.Name, &employeeCount, &item.Industry, &item.City, &item.Score); err != nil {
			return 0, err
		}
		item.IsPartner = false
		item.Description = ""
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	mu.Lock()
	res.Companies = out
	mu.Unlock()
	return len(out), nil
}

var eventTypes = map[string]struct{}{
	"webinar": {}, "conference": {}, "workshop": {}, "networking": {}, "roundtable": {}, "meetup": {}, "other": {},
}
var eventFormats = map[string]struct{}{"online": {}, "offline": {}, "hybrid": {}}
var eventStatuses = map[string]struct{}{"draft": {}, "published": {}, "cancelled": {}, "finished": {}}
var eventCurrencies = map[string]struct{}{"RUB": {}, "USD": {}, "EUR": {}, "KZT": {}, "BYN": {}}

type listEventsFilter struct {
	Type         string
	Format       string
	Mode         string
	From         *time.Time
	To           *time.Time
	Query        string
	Limit        int
	BeforeCursor int64
}

func newPublicEventID() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "evt_" + hex.EncodeToString(b), nil
}

func isValidEventPublicID(s string) bool {
	if len(s) == 13 && strings.HasPrefix(s, "evt") {
		for _, ch := range s[3:] {
			if (ch < 'a' || ch > 'f') && (ch < '0' || ch > '9') {
				return false
			}
		}
		return true
	}
	if len(s) != 16 || !strings.HasPrefix(s, "evt_") {
		return false
	}
	for _, ch := range s[4:] {
		if (ch < 'a' || ch > 'f') && (ch < '0' || ch > '9') {
			return false
		}
	}
	return true
}

func parseEventTime(s string, tz string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.FixedZone("MSK", 3*3600)
	}
	t, err := time.ParseInLocation("2006-01-02T15:04", s, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: неверный формат времени", errValidation)
	}
	return t.UTC(), nil
}

func validateEvent(req *createEventRequest, userID int64, db *sql.DB) (time.Time, time.Time, error) {
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	req.City = strings.TrimSpace(req.City)
	req.OnlineURL = strings.TrimSpace(req.OnlineURL)
	req.Timezone = strings.TrimSpace(req.Timezone)
	if req.Timezone == "" {
		req.Timezone = "Europe/Moscow"
	}
	if len(req.Title) < 3 || len(req.Title) > 200 {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: название 3..200 символов", errValidation)
	}
	if len(req.Description) > 10000 {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: описание слишком длинное", errValidation)
	}
	if req.Type == "" {
		req.Type = "webinar"
	}
	if _, ok := eventTypes[req.Type]; !ok {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: некорректный type", errValidation)
	}
	if req.Format == "" {
		req.Format = "online"
	}
	if _, ok := eventFormats[req.Format]; !ok {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: некорректный format", errValidation)
	}
	startsAt, err := parseEventTime(req.StartsAt, req.Timezone)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	endsAt, err := parseEventTime(req.EndsAt, req.Timezone)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if endsAt.Before(startsAt) {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: ends_at должен быть >= starts_at", errValidation)
	}
	if req.Format != "online" && len(req.City) < 1 {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: для офлайн/гибрид нужен city", errValidation)
	}
	if req.Format != "offline" {
		if !strings.HasPrefix(req.OnlineURL, "https://") || len(req.OnlineURL) > 500 {
			return time.Time{}, time.Time{}, fmt.Errorf("%w: online_url должен начинаться с https://", errValidation)
		}
	}
	if req.FeeCents < 0 || req.FeeCents > 100_000_000 {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: некорректная стоимость", errValidation)
	}
	if req.Currency == "" {
		req.Currency = "RUB"
	}
	if _, ok := eventCurrencies[req.Currency]; !ok {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: некорректная валюта", errValidation)
	}
	if req.SeatsTotal < 0 || req.SeatsTotal > 100000 {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: некорректное количество мест", errValidation)
	}
	tags, err := validateTags(req.Tags)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	req.Tags = tags
	if req.CommunityID != nil {
		var role string
		err := db.QueryRow(`SELECT role FROM community_members WHERE community_id=$1 AND user_id=$2`, *req.CommunityID, userID).Scan(&role)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return time.Time{}, time.Time{}, fmt.Errorf("%w: нет прав в сообществе", errForbidden)
			}
			return time.Time{}, time.Time{}, err
		}
		if role != "owner" && role != "admin" {
			return time.Time{}, time.Time{}, fmt.Errorf("%w: нет прав в сообществе", errForbidden)
		}
	}
	return startsAt, endsAt, nil
}

func handleEventError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, errForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, errNotFound), errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, "Мероприятие не найдено")
	case errors.Is(err, errConflict):
		writeError(w, http.StatusConflict, err.Error())
	default:
		log.Printf("event error: %v", err)
		writeError(w, http.StatusInternalServerError, "Ошибка сервера")
	}
}

func extractPathParam(pathValue, prefix string) (string, bool) {
	if !strings.HasPrefix(pathValue, prefix) {
		return "", false
	}
	v := strings.Trim(strings.TrimPrefix(pathValue, prefix), "/")
	if v == "" {
		return "", false
	}
	return v, true
}

func isValidPostPublicID(s string) bool {
	if len(s) != 13 || !strings.HasPrefix(s, "p") {
		return false
	}
	for _, ch := range s[1:] {
		if (ch < 'a' || ch > 'f') && (ch < '0' || ch > '9') {
			return false
		}
	}
	return true
}

func isValidUserPublicID(s string) bool {
	if len(s) != 11 || !strings.HasPrefix(s, "u") {
		return false
	}
	for _, ch := range s[1:] {
		if (ch < 'a' || ch > 'f') && (ch < '0' || ch > '9') {
			return false
		}
	}
	return true
}

func handlePostActionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, errConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, errNotFound), errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, "Не найдено")
	default:
		log.Printf("post action error: %v", err)
		writeError(w, http.StatusInternalServerError, "Ошибка сервера")
	}
}

func eventFilterFromRequest(r *http.Request) listEventsFilter {
	q := r.URL.Query()
	f := listEventsFilter{
		Type:         strings.TrimSpace(q.Get("type")),
		Format:       strings.TrimSpace(q.Get("format")),
		Mode:         strings.TrimSpace(q.Get("mode")),
		Query:        strings.TrimSpace(q.Get("q")),
		Limit:        parseLimit(q.Get("limit"), 20, 100),
		BeforeCursor: parseIDOrZero(q.Get("before_id")),
	}
	if f.Mode == "" {
		f.Mode = "all"
	}
	if raw := strings.TrimSpace(q.Get("from")); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			u := t.UTC()
			f.From = &u
		}
	}
	if raw := strings.TrimSpace(q.Get("to")); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			u := t.UTC()
			f.To = &u
		}
	}
	return f
}

func scanEventRow(row scanner) (event, error) {
	var item event
	var tagsJSON []byte
	var communityName sql.NullString
	var communityID sql.NullInt64
	err := row.Scan(
		&item.ID, &item.PublicID, &item.OrganizerID, &item.OrganizerPublic, &item.OrganizerName,
		&communityID, &communityName, &item.Title, &item.Description, &item.Type, &item.Format, &item.Category,
		&item.City, &item.Address, &item.Venue, &item.OnlineURL, &item.StartsAt, &item.EndsAt, &item.Timezone,
		&item.CoverURL, &item.BannerColor, &item.FeeCents, &item.Currency, &item.SeatsTotal, &item.RegisteredCount,
		&item.SavedCount, &item.ViewsCount, &tagsJSON, &item.Status, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return event{}, err
	}
	if communityID.Valid {
		id := communityID.Int64
		item.CommunityID = &id
	}
	if communityName.Valid {
		item.CommunityName = communityName.String
	}
	_ = json.Unmarshal(tagsJSON, &item.Tags)
	return item, nil
}

type scanner interface{ Scan(dest ...any) error }

func createEvent(db *sql.DB, organizerID int64, req createEventRequest) (event, error) {
	startsAt, endsAt, err := validateEvent(&req, organizerID, db)
	if err != nil {
		return event{}, err
	}
	var pgTags pgtype.FlatArray[string] = req.Tags
	for i := 0; i < 5; i++ {
		pid, err := newPublicEventID()
		if err != nil {
			return event{}, err
		}
		row := db.QueryRow(`
			INSERT INTO events (public_id, organizer_id, community_id, title, description, type, format, category, city, address, venue, online_url, starts_at, ends_at, timezone, cover_url, banner_color, fee_cents, currency, seats_total, tags, status)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,'published')
			RETURNING id, public_id, organizer_id,
			  (SELECT public_id FROM users WHERE id=organizer_id),
			  (SELECT full_name FROM users WHERE id=organizer_id),
			  community_id, (SELECT name FROM communities WHERE id=community_id),
			  title, description, type, format, category, city, address, venue, online_url, starts_at, ends_at, timezone,
			  cover_url, banner_color, fee_cents, currency, seats_total, registered_count, saved_count, views_count,
			  COALESCE(array_to_json(tags), '[]'::json), status, created_at, updated_at
		`, pid, organizerID, req.CommunityID, req.Title, req.Description, req.Type, req.Format, req.Category, req.City, req.Address, req.Venue, req.OnlineURL, startsAt, endsAt, req.Timezone, req.CoverURL, req.BannerColor, req.FeeCents, req.Currency, req.SeatsTotal, pgTags)
		item, err := scanEventRow(row)
		if err == nil {
			item.IsMine = true
			return item, nil
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			continue
		}
		return event{}, err
	}
	return event{}, fmt.Errorf("failed to allocate event public_id")
}

func getEvent(db *sql.DB, publicID string, viewerID int64, hasAuth bool) (event, error) {
	item, err := getEventNoView(db, publicID, viewerID, hasAuth)
	if err != nil {
		return event{}, err
	}
	_, _ = db.Exec(`UPDATE events SET views_count = views_count + 1 WHERE id=$1`, item.ID)
	item.ViewsCount++
	return item, nil
}

func getEventNoView(db *sql.DB, publicID string, viewerID int64, hasAuth bool) (event, error) {
	row := db.QueryRow(`
		SELECT e.id, e.public_id, e.organizer_id, u.public_id, u.full_name, e.community_id, COALESCE(c.name,''),
		 e.title, e.description, e.type, e.format, e.category, e.city, e.address, e.venue, e.online_url,
		 e.starts_at, e.ends_at, e.timezone, e.cover_url, e.banner_color, e.fee_cents, e.currency, e.seats_total,
		 e.registered_count, e.saved_count, e.views_count, COALESCE(array_to_json(e.tags), '[]'::json), e.status, e.created_at, e.updated_at
		FROM events e
		JOIN users u ON u.id=e.organizer_id
		LEFT JOIN communities c ON c.id=e.community_id
		WHERE e.public_id=$1 AND e.is_deleted=FALSE AND (e.status='published' OR e.organizer_id=$2)
	`, publicID, viewerID)
	item, err := scanEventRow(row)
	if err != nil {
		return event{}, err
	}
	if hasAuth {
		_ = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM event_registrations WHERE event_id=$1 AND user_id=$2 AND status='confirmed')`, item.ID, viewerID).Scan(&item.IsRegistered)
		_ = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM event_saves WHERE event_id=$1 AND user_id=$2)`, item.ID, viewerID).Scan(&item.IsSaved)
		item.IsMine = item.OrganizerID == viewerID
	}
	return item, nil
}

func listEvents(db *sql.DB, viewerID int64, hasAuth bool, f listEventsFilter) ([]event, int64, error) {
	args := []any{}
	where := []string{"e.is_deleted=FALSE", "e.status='published'"}
	if f.Mode == "organized" {
		where = []string{"e.is_deleted=FALSE", "e.organizer_id=$1"}
		args = append(args, viewerID)
	} else if f.Mode == "mine" {
		where = append(where, "EXISTS(SELECT 1 FROM event_registrations er WHERE er.event_id=e.id AND er.user_id=$1 AND er.status='confirmed')")
		args = append(args, viewerID)
	} else if f.Mode == "saved" {
		where = append(where, "EXISTS(SELECT 1 FROM event_saves es WHERE es.event_id=e.id AND es.user_id=$1)")
		args = append(args, viewerID)
	}
	if f.Type != "" {
		args = append(args, f.Type)
		where = append(where, fmt.Sprintf("e.type=$%d", len(args)))
	}
	if f.Format != "" {
		args = append(args, f.Format)
		where = append(where, fmt.Sprintf("e.format=$%d", len(args)))
	}
	if f.From != nil {
		args = append(args, *f.From)
		where = append(where, fmt.Sprintf("e.starts_at >= $%d", len(args)))
	}
	if f.To != nil {
		args = append(args, *f.To)
		where = append(where, fmt.Sprintf("e.starts_at <= $%d", len(args)))
	}
	if f.Query != "" {
		args = append(args, "%"+chatEscapeILike(f.Query)+"%")
		where = append(where, fmt.Sprintf("(e.title ILIKE $%d ESCAPE '\\' OR e.description ILIKE $%d ESCAPE '\\')", len(args), len(args)))
	}
	if f.From == nil && f.To == nil && f.Mode == "all" {
		where = append(where, "e.starts_at >= NOW() - INTERVAL '1 day'")
	}
	args = append(args, f.Limit)
	rows, err := db.Query(`
		SELECT e.id, e.public_id, e.organizer_id, u.public_id, u.full_name, e.community_id, COALESCE(c.name,''),
		 e.title, e.description, e.type, e.format, e.category, e.city, e.address, e.venue, e.online_url,
		 e.starts_at, e.ends_at, e.timezone, e.cover_url, e.banner_color, e.fee_cents, e.currency, e.seats_total,
		 e.registered_count, e.saved_count, e.views_count, COALESCE(array_to_json(e.tags), '[]'::json), e.status, e.created_at, e.updated_at
		FROM events e
		JOIN users u ON u.id=e.organizer_id
		LEFT JOIN communities c ON c.id=e.community_id
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY e.starts_at ASC, e.id DESC
		LIMIT $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []event
	for rows.Next() {
		item, err := scanEventRow(rows)
		if err != nil {
			return nil, 0, err
		}
		if hasAuth {
			_ = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM event_registrations WHERE event_id=$1 AND user_id=$2 AND status='confirmed')`, item.ID, viewerID).Scan(&item.IsRegistered)
			_ = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM event_saves WHERE event_id=$1 AND user_id=$2)`, item.ID, viewerID).Scan(&item.IsSaved)
			item.IsMine = item.OrganizerID == viewerID
		}
		items = append(items, item)
	}
	return items, 0, nil
}

func updateEvent(db *sql.DB, userID int64, publicID string, req updateEventRequest) (event, error) {
	current, err := getEventNoView(db, publicID, userID, true)
	if err != nil {
		return event{}, err
	}
	if current.OrganizerID != userID {
		return event{}, fmt.Errorf("%w: нельзя редактировать чужое мероприятие", errForbidden)
	}
	if req.Title != nil {
		current.Title = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		current.Description = strings.TrimSpace(*req.Description)
	}
	if req.Type != nil {
		current.Type = *req.Type
	}
	if req.Format != nil {
		current.Format = *req.Format
	}
	if req.Category != nil {
		current.Category = *req.Category
	}
	if req.City != nil {
		current.City = *req.City
	}
	if req.Address != nil {
		current.Address = *req.Address
	}
	if req.Venue != nil {
		current.Venue = *req.Venue
	}
	if req.OnlineURL != nil {
		current.OnlineURL = *req.OnlineURL
	}
	if req.Timezone != nil {
		current.Timezone = *req.Timezone
	}
	if req.CoverURL != nil {
		current.CoverURL = *req.CoverURL
	}
	if req.BannerColor != nil {
		current.BannerColor = *req.BannerColor
	}
	if req.FeeCents != nil {
		current.FeeCents = *req.FeeCents
	}
	if req.Currency != nil {
		current.Currency = *req.Currency
	}
	if req.SeatsTotal != nil {
		current.SeatsTotal = *req.SeatsTotal
	}
	if req.Status != nil {
		if _, ok := eventStatuses[*req.Status]; !ok {
			return event{}, fmt.Errorf("%w: некорректный status", errValidation)
		}
		current.Status = *req.Status
	}
	if req.Tags != nil {
		current.Tags = *req.Tags
	}
	if req.StartsAt != nil {
		current.StartsAt, _ = parseEventTime(*req.StartsAt, current.Timezone)
	}
	if req.EndsAt != nil {
		current.EndsAt, _ = parseEventTime(*req.EndsAt, current.Timezone)
	}
	var pgTags pgtype.FlatArray[string] = current.Tags
	_, err = db.Exec(`UPDATE events SET title=$1, description=$2, type=$3, format=$4, category=$5, city=$6, address=$7, venue=$8, online_url=$9, starts_at=$10, ends_at=$11, timezone=$12, cover_url=$13, banner_color=$14, fee_cents=$15, currency=$16, seats_total=$17, tags=$18, status=$19, updated_at=NOW() WHERE public_id=$20 AND organizer_id=$21`,
		current.Title, current.Description, current.Type, current.Format, current.Category, current.City, current.Address, current.Venue, current.OnlineURL, current.StartsAt, current.EndsAt, current.Timezone, current.CoverURL, current.BannerColor, current.FeeCents, current.Currency, current.SeatsTotal, pgTags, current.Status, publicID, userID)
	if err != nil {
		return event{}, err
	}
	return getEventNoView(db, publicID, userID, true)
}

func deleteEvent(db *sql.DB, userID int64, publicID string) error {
	res, err := db.Exec(`UPDATE events SET is_deleted=TRUE, status='cancelled', updated_at=NOW() WHERE public_id=$1 AND organizer_id=$2`, publicID, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errForbidden
	}
	return nil
}

func registerToEvent(db *sql.DB, userID int64, publicID, ticketType string) (string, int, error) {
	if len(ticketType) > 32 {
		return "", 0, fmt.Errorf("%w: ticket_type слишком длинный", errValidation)
	}
	if ticketType == "" {
		ticketType = "standard"
	}
	tx, err := db.Begin()
	if err != nil {
		return "", 0, err
	}
	defer tx.Rollback()
	var eventID int64
	var seatsTotal, regCount int
	err = tx.QueryRow(`SELECT id, seats_total, registered_count FROM events WHERE public_id=$1 AND is_deleted=FALSE AND status='published' AND ends_at > NOW()`, publicID).Scan(&eventID, &seatsTotal, &regCount)
	if err != nil {
		return "", 0, errNotFound
	}
	status := "confirmed"
	if seatsTotal > 0 && regCount >= seatsTotal {
		status = "waitlist"
	}
	_, err = tx.Exec(`INSERT INTO event_registrations(event_id,user_id,ticket_type,status) VALUES ($1,$2,$3,$4) ON CONFLICT (event_id,user_id) DO UPDATE SET status=EXCLUDED.status, ticket_type=EXCLUDED.ticket_type`, eventID, userID, ticketType, status)
	if err != nil {
		return "", 0, err
	}
	var count int
	if err := tx.QueryRow(`UPDATE events SET registered_count=(SELECT COUNT(*) FROM event_registrations WHERE event_id=$1 AND status='confirmed') WHERE id=$1 RETURNING registered_count`, eventID).Scan(&count); err != nil {
		return "", 0, err
	}
	if err := tx.Commit(); err != nil {
		return "", 0, err
	}
	return status, count, nil
}

func cancelEventRegistration(db *sql.DB, userID int64, publicID string) (int, error) {
	var eventID int64
	if err := db.QueryRow(`SELECT id FROM events WHERE public_id=$1`, publicID).Scan(&eventID); err != nil {
		return 0, errNotFound
	}
	_, _ = db.Exec(`UPDATE event_registrations SET status='cancelled' WHERE event_id=$1 AND user_id=$2`, eventID, userID)
	var count int
	if err := db.QueryRow(`UPDATE events SET registered_count=(SELECT COUNT(*) FROM event_registrations WHERE event_id=$1 AND status='confirmed') WHERE id=$1 RETURNING registered_count`, eventID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func saveEvent(db *sql.DB, userID int64, publicID string) (int, error) {
	var eventID int64
	if err := db.QueryRow(`SELECT id FROM events WHERE public_id=$1 AND is_deleted=FALSE`, publicID).Scan(&eventID); err != nil {
		return 0, errNotFound
	}
	_, err := db.Exec(`INSERT INTO event_saves(event_id,user_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, eventID, userID)
	if err != nil {
		return 0, err
	}
	var count int
	if err := db.QueryRow(`UPDATE events SET saved_count=(SELECT COUNT(*) FROM event_saves WHERE event_id=$1) WHERE id=$1 RETURNING saved_count`, eventID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func unsaveEvent(db *sql.DB, userID int64, publicID string) (int, error) {
	var eventID int64
	if err := db.QueryRow(`SELECT id FROM events WHERE public_id=$1`, publicID).Scan(&eventID); err != nil {
		return 0, errNotFound
	}
	_, err := db.Exec(`DELETE FROM event_saves WHERE event_id=$1 AND user_id=$2`, eventID, userID)
	if err != nil {
		return 0, err
	}
	var count int
	if err := db.QueryRow(`UPDATE events SET saved_count=(SELECT COUNT(*) FROM event_saves WHERE event_id=$1) WHERE id=$1 RETURNING saved_count`, eventID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func listEventsByDate(db *sql.DB, year, month int, viewerID int64, hasAuth bool) ([]event, error) {
	from := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0).Add(-time.Second)
	items, _, err := listEvents(db, viewerID, hasAuth, listEventsFilter{From: &from, To: &to, Limit: 200, Mode: "all"})
	return items, err
}

func listEventRegistrations(db *sql.DB, userID int64, publicID string, limit int) ([]map[string]any, error) {
	var eventID int64
	var organizerID int64
	if err := db.QueryRow(`SELECT id, organizer_id FROM events WHERE public_id=$1`, publicID).Scan(&eventID, &organizerID); err != nil {
		return nil, errNotFound
	}
	if organizerID != userID {
		return nil, errForbidden
	}
	rows, err := db.Query(`SELECT u.public_id, u.full_name, er.ticket_type, er.status, er.created_at FROM event_registrations er JOIN users u ON u.id=er.user_id WHERE er.event_id=$1 ORDER BY er.created_at DESC LIMIT $2`, eventID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var pub, name, ticket, status string
		var created time.Time
		if err := rows.Scan(&pub, &name, &ticket, &status, &created); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"user_public_id": pub, "full_name": name, "ticket_type": ticket, "status": status, "created_at": created})
	}
	return out, nil
}

func migrateEventSeedData(db *sql.DB) error {
	const seedUserEmail = "seed.lastop@local"

	var organizerID int64
	err := db.QueryRow(`SELECT id FROM users WHERE email = $1`, seedUserEmail).Scan(&organizerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("lookup seed organizer: %w", err)
	}

	type eventSeed struct {
		PublicID    string
		Title       string
		Description string
		Type        string
		Category    string
		Format      string
		City        string
		Address     string
		Venue       string
		OnlineURL   string
		StartsIn    time.Duration
		DurationMin int
		Timezone    string
		FeeCents    int
		Currency    string
		SeatsTotal  int
		Tags        []string
		CoverURL    string
		BannerColor string
		Status      string
	}

	seeds := []eventSeed{
		{
			PublicID:    "evtseed1a2b3c",
			Title:       "Запуск платформы LASTOP",
			Description: "Демо-вебинар к запуску платформы. Покажем основные возможности и ответим на вопросы.",
			Type:        "webinar",
			Category:    "Платформа",
			Format:      "online",
			City:        "",
			Address:     "",
			Venue:       "",
			OnlineURL:   "https://meet.lastop.example/launch",
			StartsIn:    48 * time.Hour,
			DurationMin: 60,
			Timezone:    "Europe/Moscow",
			FeeCents:    0,
			Currency:    "RUB",
			SeatsTotal:  0,
			Tags:        []string{"вебинар", "запуск"},
			CoverURL:    "",
			BannerColor: "",
			Status:      "published",
		},
		{
			PublicID:    "evtseed4d5e6f",
			Title:       "Networking-завтрак: логистика 2026",
			Description: "Встреча для специалистов отрасли. Завтрак, кофе, обмен контактами.",
			Type:        "networking",
			Category:    "Логистика",
			Format:      "offline",
			City:        "Москва",
			Address:     "Лесная ул., 5",
			Venue:       "Лофт «Северный»",
			OnlineURL:   "",
			StartsIn:    7 * 24 * time.Hour,
			DurationMin: 120,
			Timezone:    "Europe/Moscow",
			FeeCents:    150000,
			Currency:    "RUB",
			SeatsTotal:  40,
			Tags:        []string{"нетворкинг", "москва"},
			CoverURL:    "",
			BannerColor: "",
			Status:      "published",
		},
		{
			PublicID:    "evtseed789abc",
			Title:       "Воркшоп: оптимизация маршрутов",
			Description: "Практический разбор кейсов и инструментов планирования маршрутов.",
			Type:        "workshop",
			Category:    "Транспорт",
			Format:      "hybrid",
			City:        "Санкт-Петербург",
			Address:     "Невский пр., 100",
			Venue:       "Бизнес-центр «Невский»",
			OnlineURL:   "https://meet.lastop.example/workshop",
			StartsIn:    14 * 24 * time.Hour,
			DurationMin: 180,
			Timezone:    "Europe/Moscow",
			FeeCents:    0,
			Currency:    "RUB",
			SeatsTotal:  60,
			Tags:        []string{"воркшоп", "маршруты"},
			CoverURL:    "",
			BannerColor: "",
			Status:      "published",
		},
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	for _, s := range seeds {
		starts := time.Now().Add(s.StartsIn).UTC()
		ends := starts.Add(time.Duration(s.DurationMin) * time.Minute)

		// Маппинг колонок и аргументов миграции seed-мероприятий.
		// Любое расхождение между порядком колонок и порядком аргументов
		// в Exec приведёт к runtime-ошибке кодирования pgx.
		//
		// $1  public_id      string    TEXT
		// $2  organizer_id   int64     BIGINT
		// $3  title          string    TEXT
		// $4  description    string    TEXT
		// $5  type           string    TEXT
		// $6  category       string    TEXT
		// $7  format         string    TEXT
		// $8  city           string    TEXT
		// $9  address        string    TEXT
		// $10 venue          string    TEXT
		// $11 online_url     string    TEXT
		// $12 starts_at      time.Time TIMESTAMPTZ
		// $13 ends_at        time.Time TIMESTAMPTZ
		// $14 timezone       string    TEXT
		// $15 fee_cents      int       INTEGER
		// $16 currency       string    TEXT
		// $17 seats_total    int       INTEGER
		// $18 tags           []string  TEXT[]
		// $19 cover_url      string    TEXT
		// $20 banner_color   string    TEXT
		// $21 status         string    TEXT
		_, err := tx.Exec(`
			INSERT INTO events (
				public_id, organizer_id, title, description, type,
				category, format, city, address, venue,
				online_url, starts_at, ends_at, timezone, fee_cents,
				currency, seats_total, tags, cover_url, banner_color,
				status
			) VALUES (
				$1,  $2,  $3,  $4,  $5,
				$6,  $7,  $8,  $9,  $10,
				$11, $12, $13, $14, $15,
				$16, $17, $18, $19, $20,
				$21
			)
			ON CONFLICT (public_id) DO NOTHING
		`,
			s.PublicID,
			organizerID,
			s.Title,
			s.Description,
			s.Type,
			s.Category,
			s.Format,
			s.City,
			s.Address,
			s.Venue,
			s.OnlineURL,
			starts,
			ends,
			s.Timezone,
			s.FeeCents,
			s.Currency,
			s.SeatsTotal,
			s.Tags,
			s.CoverURL,
			s.BannerColor,
			s.Status,
		)
		if err != nil {
			return fmt.Errorf("insert seed event %s: %w", s.PublicID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func handleChatError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, errForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, errNotFound), errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, errConflict):
		writeError(w, http.StatusConflict, err.Error())
	default:
		log.Printf("chat error: %v", err)
		writeError(w, http.StatusInternalServerError, "Ошибка сервера")
	}
}

func getUserIDByPublicID(db *sql.DB, publicID string) (int64, error) {
	var id int64
	err := db.QueryRow(`SELECT id FROM users WHERE public_id=$1`, publicID).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, errNotFound
		}
		return 0, err
	}
	return id, nil
}

func findOrCreateDirectConversation(db *sql.DB, userID int64, targetPublicID string) (chatConversation, error) {
	targetID, err := getUserIDByPublicID(db, targetPublicID)
	if err != nil {
		return chatConversation{}, err
	}
	if targetID == userID {
		return chatConversation{}, fmt.Errorf("%w: нельзя создать диалог с собой", errValidation)
	}
	ids := []int64{userID, targetID}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	tx, err := db.Begin()
	if err != nil {
		return chatConversation{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`SELECT pg_advisory_xact_lock($1,$2)`, ids[0], ids[1]); err != nil {
		return chatConversation{}, err
	}
	var convID int64
	err = tx.QueryRow(`SELECT c.id FROM chat_conversations c JOIN chat_participants p1 ON p1.conversation_id=c.id AND p1.user_id=$1 JOIN chat_participants p2 ON p2.conversation_id=c.id AND p2.user_id=$2 WHERE c.type='direct' LIMIT 1`, userID, targetID).Scan(&convID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return chatConversation{}, err
	}
	if errors.Is(err, sql.ErrNoRows) {
		pid, _ := newChatPublicID()
		if err := tx.QueryRow(`INSERT INTO chat_conversations(public_id,type,created_by,last_message_at) VALUES($1,'direct',$2,NOW()) RETURNING id`, pid, userID).Scan(&convID); err != nil {
			return chatConversation{}, err
		}
		if _, err := tx.Exec(`INSERT INTO chat_participants(conversation_id,user_id,role) VALUES($1,$2,'member'),($1,$3,'member')`, convID, userID, targetID); err != nil {
			return chatConversation{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return chatConversation{}, err
	}
	conv, err := getConversationByPublicID(db, userID, "")
	if err == nil {
		_ = conv
	}
	return getConversationByID(db, userID, convID)
}

func getConversationByID(db *sql.DB, userID, convID int64) (chatConversation, error) {
	var pid string
	if err := db.QueryRow(`SELECT public_id FROM chat_conversations WHERE id=$1`, convID).Scan(&pid); err != nil {
		return chatConversation{}, errNotFound
	}
	return getConversationByPublicID(db, userID, pid)
}

func getConversationByPublicID(db *sql.DB, userID int64, publicID string) (chatConversation, error) {
	var c chatConversation
	var lastAuthorID sql.NullInt64
	var lastAuthorName, lastContent string
	var pinned, muted bool
	var role string
	var otherID sql.NullString
	var otherName, otherPosition, otherCompany, otherAvatar string
	var otherUID sql.NullInt64
	err := db.QueryRow(`
SELECT c.id,c.public_id,c.type,c.title,c.avatar_url,c.community_id,c.created_at,c.last_message_at,
       COALESCE(p.pinned,false),COALESCE(p.muted,false),COALESCE(p.role,'member'),
       (SELECT COUNT(*) FROM chat_participants cp WHERE cp.conversation_id=c.id),
       (SELECT COUNT(*) FROM chat_messages m WHERE m.conversation_id=c.id AND m.id>p.last_read_message_id AND m.author_id<>$1 AND m.is_deleted=FALSE),
       COALESCE((SELECT m.content FROM chat_messages m WHERE m.conversation_id=c.id ORDER BY m.id DESC LIMIT 1),''),
       (SELECT m.author_id FROM chat_messages m WHERE m.conversation_id=c.id ORDER BY m.id DESC LIMIT 1),
       COALESCE((SELECT u.full_name FROM chat_messages m JOIN users u ON u.id=m.author_id WHERE m.conversation_id=c.id ORDER BY m.id DESC LIMIT 1),''),
       COALESCE((SELECT u2.public_id FROM chat_participants cp2 JOIN users u2 ON u2.id=cp2.user_id WHERE cp2.conversation_id=c.id AND cp2.user_id<>$1 LIMIT 1),''),
       COALESCE((SELECT u2.id FROM chat_participants cp2 JOIN users u2 ON u2.id=cp2.user_id WHERE cp2.conversation_id=c.id AND cp2.user_id<>$1 LIMIT 1),0),
       COALESCE((SELECT u2.full_name FROM chat_participants cp2 JOIN users u2 ON u2.id=cp2.user_id WHERE cp2.conversation_id=c.id AND cp2.user_id<>$1 LIMIT 1),''),
       COALESCE((SELECT u2.position FROM chat_participants cp2 JOIN users u2 ON u2.id=cp2.user_id WHERE cp2.conversation_id=c.id AND cp2.user_id<>$1 LIMIT 1),''),
       COALESCE((SELECT u2.company_name FROM chat_participants cp2 JOIN users u2 ON u2.id=cp2.user_id WHERE cp2.conversation_id=c.id AND cp2.user_id<>$1 LIMIT 1),''),
       COALESCE((SELECT u2.avatar_url FROM chat_participants cp2 JOIN users u2 ON u2.id=cp2.user_id WHERE cp2.conversation_id=c.id AND cp2.user_id<>$1 LIMIT 1),'')
FROM chat_conversations c JOIN chat_participants p ON p.conversation_id=c.id
WHERE p.user_id=$1 AND c.public_id=$2`, userID, publicID).Scan(&c.ID, &c.PublicID, &c.Type, &c.Title, &c.AvatarURL, &c.CommunityID, &c.CreatedAt, &c.LastMessageAt, &pinned, &muted, &role, &c.MembersCount, &c.UnreadCount, &lastContent, &lastAuthorID, &lastAuthorName, &otherID, &otherUID, &otherName, &otherPosition, &otherCompany, &otherAvatar)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c, errNotFound
		}
		return c, err
	}
	c.Pinned, c.Muted, c.MyRole = pinned, muted, role
	if c.Type == "direct" {
		c.DisplayName = otherName
		c.DisplayRole = strings.TrimSpace(strings.TrimSpace(otherPosition+" · ") + otherCompany)
		c.DisplayAvatar = otherAvatar
		c.OtherPublicID = otherID.String
		c.DisplayColor = stableColorForName(otherName)
	} else {
		c.DisplayName = c.Title
		c.DisplayAvatar = c.AvatarURL
		c.DisplayColor = stableColorForName(c.Title)
	}
	c.LastMessageText = messagePreview(lastContent, 120)
	if lastAuthorID.Valid && lastAuthorID.Int64 == userID {
		c.LastMessageAuthor = "Вы"
	} else {
		c.LastMessageAuthor = lastAuthorName
	}
	return c, nil
}

func listConversations(db *sql.DB, userID int64, filter, q string, limit int) ([]chatConversation, error) {
	rows, err := db.Query(`SELECT c.public_id FROM chat_conversations c JOIN chat_participants p ON p.conversation_id=c.id WHERE p.user_id=$1 ORDER BY p.pinned DESC, c.last_message_at DESC NULLS LAST, c.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]chatConversation, 0)
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			return nil, err
		}
		c, err := getConversationByPublicID(db, userID, pid)
		if err == nil {
			items = append(items, c)
		}
	}
	q = strings.ToLower(strings.TrimSpace(q))
	out := make([]chatConversation, 0, len(items))
	for _, c := range items {
		if filter == "unread" && c.UnreadCount == 0 {
			continue
		}
		if filter == "groups" && !(c.Type == "group" || c.Type == "community") {
			continue
		}
		if filter == "companies" {
			role := strings.TrimSpace(c.DisplayRole)
			isCompany := role != "" && strings.Contains(role, "·")
			if !isCompany {
				continue
			}
		}
		if q != "" && !strings.Contains(strings.ToLower(c.DisplayName+" "+c.LastMessageText), q) {
			continue
		}
		out = append(out, c)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func createGroupConversation(db *sql.DB, creatorID int64, req createGroupConversationRequest) (chatConversation, error) {
	title := strings.TrimSpace(req.Title)
	if utf8.RuneCountInString(title) < 3 || utf8.RuneCountInString(title) > 120 {
		return chatConversation{}, fmt.Errorf("%w: название группы 3..120", errValidation)
	}
	uniq := map[string]struct{}{}
	members := make([]int64, 0, len(req.MemberIDs)+1)
	members = append(members, creatorID)
	for _, pid := range req.MemberIDs {
		pid = strings.TrimSpace(pid)
		if pid == "" {
			continue
		}
		if _, ok := uniq[pid]; ok {
			continue
		}
		uniq[pid] = struct{}{}
		id, err := getUserIDByPublicID(db, pid)
		if err != nil {
			return chatConversation{}, err
		}
		if id != creatorID {
			members = append(members, id)
		}
	}
	if len(members) < 3 || len(members) > 51 {
		return chatConversation{}, fmt.Errorf("%w: участников 2..50 кроме создателя", errValidation)
	}
	tx, err := db.Begin()
	if err != nil {
		return chatConversation{}, err
	}
	defer tx.Rollback()
	pid, _ := newChatPublicID()
	var convID int64
	if err := tx.QueryRow(`INSERT INTO chat_conversations(public_id,type,title,created_by,last_message_at) VALUES($1,'group',$2,$3,NOW()) RETURNING id`, pid, title, creatorID).Scan(&convID); err != nil {
		return chatConversation{}, err
	}
	for i, uid := range members {
		role := "member"
		if i == 0 {
			role = "owner"
		}
		if _, err := tx.Exec(`INSERT INTO chat_participants(conversation_id,user_id,role) VALUES($1,$2,$3)`, convID, uid, role); err != nil {
			return chatConversation{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return chatConversation{}, err
	}
	return getConversationByID(db, creatorID, convID)
}

func getOrCreateCommunityConversation(db *sql.DB, communityID int64, userID int64) (chatConversation, error) {
	var isMember bool
	if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM community_members WHERE community_id=$1 AND user_id=$2)`, communityID, userID).Scan(&isMember); err != nil {
		return chatConversation{}, err
	}
	if !isMember {
		return chatConversation{}, fmt.Errorf("%w: вы не участник сообщества", errForbidden)
	}
	var convID int64
	err := db.QueryRow(`SELECT id FROM chat_conversations WHERE type='community' AND community_id=$1 LIMIT 1`, communityID).Scan(&convID)
	if errors.Is(err, sql.ErrNoRows) {
		tx, err := db.Begin()
		if err != nil {
			return chatConversation{}, err
		}
		defer tx.Rollback()
		var title, avatar string
		if err := tx.QueryRow(`SELECT name,COALESCE(avatar_url,'') FROM communities WHERE id=$1`, communityID).Scan(&title, &avatar); err != nil {
			return chatConversation{}, err
		}
		pid, _ := newChatPublicID()
		if err := tx.QueryRow(`INSERT INTO chat_conversations(public_id,type,title,avatar_url,community_id,created_by,last_message_at) VALUES($1,'community',$2,$3,$4,$5,NOW()) RETURNING id`, pid, title, avatar, communityID, userID).Scan(&convID); err != nil {
			return chatConversation{}, err
		}
		rows, err := tx.Query(`SELECT user_id,role FROM community_members WHERE community_id=$1`, communityID)
		if err != nil {
			return chatConversation{}, err
		}
		defer rows.Close()
		for rows.Next() {
			var uid int64
			var r string
			_ = rows.Scan(&uid, &r)
			role := "member"
			if r == "owner" || r == "admin" {
				role = "admin"
			}
			_, _ = tx.Exec(`INSERT INTO chat_participants(conversation_id,user_id,role) VALUES($1,$2,$3) ON CONFLICT(conversation_id,user_id) DO UPDATE SET role=EXCLUDED.role`, convID, uid, role)
		}
		if err := tx.Commit(); err != nil {
			return chatConversation{}, err
		}
	} else if err != nil {
		return chatConversation{}, err
	}
	return getConversationByID(db, userID, convID)
}

func searchInConversation(db *sql.DB, userID int64, publicID, q string) ([]chatMessage, error) {
	cid, err := ensureParticipant(db, userID, publicID)
	if err != nil {
		return nil, err
	}
	q = strings.TrimSpace(q)
	if q == "" {
		return []chatMessage{}, nil
	}
	pattern := "%" + chatEscapeILike(q) + "%"
	rows, err := db.Query(`SELECT m.id,m.conversation_id,m.content,m.is_edited,m.is_deleted,m.created_at,m.edited_at,u.public_id,u.full_name,COALESCE(u.avatar_url,''),m.attachment_url,m.attachment_name,m.attachment_type,m.attachment_size FROM chat_messages m JOIN users u ON u.id=m.author_id WHERE m.conversation_id=$1 AND m.content ILIKE $2 ESCAPE '\' ORDER BY m.id DESC LIMIT 50`, cid, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []chatMessage{}
	for rows.Next() {
		var m chatMessage
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Content, &m.IsEdited, &m.IsDeleted, &m.CreatedAt, &m.EditedAt, &m.AuthorPublicID, &m.AuthorName, &m.AuthorAvatar, &m.AttachmentURL, &m.AttachmentName, &m.AttachmentType, &m.AttachmentSize); err != nil {
			return nil, err
		}
		m.AuthorColor = stableColorForName(m.AuthorName)
		out = append(out, m)
	}
	return out, nil
}
func ensureParticipant(db *sql.DB, userID int64, conversationPublicID string) (int64, error) {
	var cid int64
	err := db.QueryRow(`SELECT c.id FROM chat_conversations c JOIN chat_participants p ON p.conversation_id=c.id WHERE c.public_id=$1 AND p.user_id=$2`, conversationPublicID, userID).Scan(&cid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, errForbidden
		}
		return 0, err
	}
	return cid, nil
}

func listMessages(db *sql.DB, userID int64, conversationPublicID string, limit int, beforeID int64) ([]chatMessage, error) {
	cid, err := ensureParticipant(db, userID, conversationPublicID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	q := `SELECT m.id,m.conversation_id,m.content,m.reply_to_id,m.is_edited,m.is_deleted,m.created_at,m.edited_at,u.public_id,u.full_name,COALESCE(u.avatar_url,''),m.attachment_url,m.attachment_name,m.attachment_type,m.attachment_size FROM chat_messages m JOIN users u ON u.id=m.author_id WHERE m.conversation_id=$1`
	args := []any{cid}
	if beforeID > 0 {
		q += ` AND m.id < $2`
		args = append(args, beforeID)
		q += ` ORDER BY m.id DESC LIMIT $3`
		args = append(args, limit)
	} else {
		q += ` ORDER BY m.id DESC LIMIT $2`
		args = append(args, limit)
	}
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var myPublicID string
	_ = db.QueryRow(`SELECT public_id FROM users WHERE id=$1`, userID).Scan(&myPublicID)
	out := make([]chatMessage, 0)
	for rows.Next() {
		var m chatMessage
		var reply sql.NullInt64
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Content, &reply, &m.IsEdited, &m.IsDeleted, &m.CreatedAt, &m.EditedAt, &m.AuthorPublicID, &m.AuthorName, &m.AuthorAvatar, &m.AttachmentURL, &m.AttachmentName, &m.AttachmentType, &m.AttachmentSize); err != nil {
			return nil, err
		}
		m.IsMine = m.AuthorPublicID == myPublicID
		m.AuthorColor = stableColorForName(m.AuthorName)
		out = append(out, m)
	}
	for i := 0; i < len(out)/2; i++ {
		out[i], out[len(out)-1-i] = out[len(out)-1-i], out[i]
	}
	return out, nil
}

func sendMessage(db *sql.DB, userID int64, conversationPublicID string, req sendMessageRequest) (chatMessage, error) {
	cid, err := ensureParticipant(db, userID, conversationPublicID)
	if err != nil {
		return chatMessage{}, err
	}
	content := strings.TrimSpace(req.Content)
	contentLen := utf8.RuneCountInString(content)
	hasAttachment := strings.TrimSpace(req.AttachmentURL) != ""

	// Валидация: либо текст 1..8000, либо вложение, либо и то и другое
	if !hasAttachment {
		if contentLen < 1 || contentLen > 8000 {
			return chatMessage{}, fmt.Errorf("%w: длина сообщения 1..8000", errValidation)
		}
	} else {
		if contentLen > 8000 {
			return chatMessage{}, fmt.Errorf("%w: длина сообщения до 8000", errValidation)
		}
		// Дополнительная защита — URL должен начинаться с /uploads/
		if !strings.HasPrefix(req.AttachmentURL, "/uploads/") {
			return chatMessage{}, fmt.Errorf("%w: некорректный attachment_url", errValidation)
		}
	}

	// Нормализуем поля вложения
	aURL := strings.TrimSpace(req.AttachmentURL)
	aName := strings.TrimSpace(req.AttachmentName)
	aType := strings.TrimSpace(req.AttachmentType)
	if len(aName) > 255 {
		aName = aName[:255]
	}
	if len(aType) > 100 {
		aType = aType[:100]
	}

	var mid int64
	err = db.QueryRow(`
		INSERT INTO chat_messages(conversation_id,author_id,content,reply_to_id,attachment_url,attachment_name,attachment_type,attachment_size)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
		cid, userID, content, req.ReplyToID, aURL, aName, aType, req.AttachmentSize).Scan(&mid)
	if err != nil {
		return chatMessage{}, err
	}
	_, _ = db.Exec(`UPDATE chat_conversations SET last_message_at=NOW() WHERE id=$1`, cid)
	_, _ = db.Exec(`DELETE FROM chat_typing WHERE conversation_id=$1 AND user_id=$2`, cid, userID)
	var m chatMessage
	if err := db.QueryRow(`
		SELECT m.id,m.conversation_id,m.content,m.is_edited,m.is_deleted,m.created_at,m.edited_at,
		       u.public_id,u.full_name,COALESCE(u.avatar_url,''),
		       m.attachment_url,m.attachment_name,m.attachment_type,m.attachment_size
		FROM chat_messages m JOIN users u ON u.id=m.author_id WHERE m.id=$1`, mid).Scan(
		&m.ID, &m.ConversationID, &m.Content, &m.IsEdited, &m.IsDeleted, &m.CreatedAt, &m.EditedAt,
		&m.AuthorPublicID, &m.AuthorName, &m.AuthorAvatar,
		&m.AttachmentURL, &m.AttachmentName, &m.AttachmentType, &m.AttachmentSize,
	); err != nil {
		return chatMessage{}, err
	}
	m.IsMine = true
	m.AuthorColor = stableColorForName(m.AuthorName)

	// Уведомление получателю — best-effort, после основной операции.
	// Только для direct-чатов (1-на-1). Для групповых/community-чатов не уведомляем,
	// чтобы не спамить участников.
	go notifyOnChatMessage(db, cid, conversationPublicID, userID, content)

	return m, nil
}

// notifyOnChatMessage отправляет уведомление получателю в direct-чате.
// Для групповых/community-чатов не уведомляет (избегаем спама).
// Запускать в горутине после основной операции.
func notifyOnChatMessage(db *sql.DB, conversationID int64, conversationPublicID string, actorID int64, content string) {
	// 1. Проверяем тип чата — только direct
	var chatType string
	if err := db.QueryRow(`SELECT type FROM chat_conversations WHERE id = $1`, conversationID).Scan(&chatType); err != nil {
		log.Printf("notif chat: get type: %v", err)
		return
	}
	if chatType != "direct" {
		return // групповые/community не уведомляем
	}

	// 2. Достаём ID собеседника
	var recipientID int64
	if err := db.QueryRow(`
		SELECT user_id FROM chat_participants
		WHERE conversation_id = $1 AND user_id <> $2
		LIMIT 1
	`, conversationID, actorID).Scan(&recipientID); err != nil {
		log.Printf("notif chat: get recipient: %v", err)
		return
	}
	if recipientID == 0 || recipientID == actorID {
		return
	}

	// 3. Создаём уведомление
	actorName := getUserDisplayName(db, actorID)
	title := actorName + " написал вам"
	preview := truncateRunes(content, 200)
	if err := createNotification(db, createNotificationParams{
		RecipientID:    recipientID,
		ActorID:        actorID,
		Type:           "chat_message",
		SourceType:     "chat",
		SourceID:       conversationID,
		SourcePublicID: conversationPublicID,
		Title:          title,
		Preview:        preview,
	}); err != nil {
		log.Printf("notif chat_message: %v", err)
	}
}

func setTyping(db *sql.DB, userID int64, conversationPublicID string, isTyping bool) error {
	cid, err := ensureParticipant(db, userID, conversationPublicID)
	if err != nil {
		return err
	}
	_, _ = db.Exec(`DELETE FROM chat_typing WHERE started_at < NOW() - INTERVAL '15 seconds'`)
	if isTyping {
		_, err = db.Exec(`INSERT INTO chat_typing(conversation_id,user_id,started_at) VALUES($1,$2,NOW()) ON CONFLICT(conversation_id,user_id) DO UPDATE SET started_at=EXCLUDED.started_at`, cid, userID)
		return err
	}
	_, err = db.Exec(`DELETE FROM chat_typing WHERE conversation_id=$1 AND user_id=$2`, cid, userID)
	return err
}

func listTyping(db *sql.DB, userID int64, conversationPublicID string) ([]map[string]any, error) {
	cid, err := ensureParticipant(db, userID, conversationPublicID)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT u.public_id,u.full_name FROM chat_typing t JOIN users u ON u.id=t.user_id WHERE t.conversation_id=$1 AND t.user_id<>$2 AND t.started_at > NOW() - INTERVAL '10 seconds'`, cid, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var pid, name string
		if err := rows.Scan(&pid, &name); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"public_id": pid, "name": name})
	}
	return out, nil
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func isValidEmail(s string) bool {
	addr, err := mail.ParseAddress(s)
	return err == nil && addr.Address == s
}

func securityHeaders(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.ServeHTTP(w, r)
	})
}

// gzipResponseWriter оборачивает http.ResponseWriter и пишет в gzip.Writer.
// Перехватывает WriteHeader чтобы убрать Content-Length (он будет неверным
// после сжатия) и добавить Content-Encoding/Vary.
type gzipResponseWriter struct {
	http.ResponseWriter
	gw          *gzip.Writer
	headersSent bool
}

func (g *gzipResponseWriter) WriteHeader(status int) {
	if !g.headersSent {
		g.headersSent = true
		h := g.ResponseWriter.Header()
		h.Del("Content-Length")
		h.Set("Content-Encoding", "gzip")
		h.Add("Vary", "Accept-Encoding")
	}
	g.ResponseWriter.WriteHeader(status)
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	if !g.headersSent {
		g.WriteHeader(http.StatusOK)
	}
	return g.gw.Write(b)
}

func (g *gzipResponseWriter) Flush() {
	if g.gw != nil {
		_ = g.gw.Flush()
	}
	if f, ok := g.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

var skipGzipExt = map[string]struct{}{
	".png": {}, ".jpg": {}, ".jpeg": {}, ".webp": {}, ".gif": {},
	".ico": {}, ".bmp": {}, ".tif": {}, ".tiff": {},
	".mp4": {}, ".webm": {}, ".mp3": {}, ".wav": {}, ".ogg": {},
	".zip": {}, ".gz": {}, ".rar": {}, ".7z": {},
	".woff": {}, ".woff2": {},
	".pdf": {},
}

func shouldSkipGzip(p string) bool {
	dot := strings.LastIndex(p, ".")
	if dot < 0 {
		return false
	}
	ext := strings.ToLower(p[dot:])
	_, ok := skipGzipExt[ext]
	return ok
}

func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		if shouldSkipGzip(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			next.ServeHTTP(w, r)
			return
		}

		gz := gzip.NewWriter(w)
		defer gz.Close()

		gw := &gzipResponseWriter{
			ResponseWriter: w,
			gw:             gz,
		}
		next.ServeHTTP(gw, r)
	})
}

func staticSecurity(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := path.Clean("/" + r.URL.Path)
		for _, segment := range strings.Split(clean, "/") {
			if strings.HasPrefix(segment, ".") && segment != "." && segment != ".." {
				http.NotFound(w, r)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

// injectHTML возвращает http.Handler, который оборачивает next и для
// HTML-ответов вставляет htmlInject перед </head>. Для всех остальных
// ответов (CSS, JS, PNG, JSON API) ничего не меняет — пропускает as-is.
//
// staticCacheControl выставляет корректные Cache-Control заголовки для
// статических ответов. Без этого браузеры применяют эвристическое
// кеширование (heuristic freshness ~10% от возраста файла), из-за чего
// обновлённые ассеты не доходят до пользователя после деплоя минутами
// или часами.
//
// Стратегия:
//   - HTML/CSS/JS  → "no-cache, must-revalidate"
//     Браузер хранит копию, но перед каждым использованием делает
//     условный запрос (If-None-Match / If-Modified-Since). Сервер
//     отвечает 304 Not Modified если файл не менялся — это быстро
//     и не грузит сеть. Если файл изменился — приходит новая версия
//     сразу после первого F5, без Ctrl+Shift+R.
//   - Изображения и шрифты → "public, max-age=300, must-revalidate"
//     5 минут жёсткого кеша. Картинки меняются редко, экономим RTT.
//   - API и всё прочее → ничего не выставляем (API сам ставит).
//
// Также блокируем кеширование Service Worker'ом если он когда-то
// заведётся — заголовок Vary: Cookie помогает CDN правильно
// сегментировать кеш по аутентификации.
func staticCacheControl(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// API-эндпоинты — не трогаем, сами знают что нужно
		if strings.HasPrefix(r.URL.Path, "/api/") {
			h.ServeHTTP(w, r)
			return
		}

		ext := strings.ToLower(path.Ext(r.URL.Path))
		switch ext {
		case "", ".html", ".htm", ".js", ".mjs", ".css", ".json", ".xml", ".txt", ".map":
			// Часто меняющиеся текстовые ассеты + HTML без расширения.
			// no-cache + must-revalidate = браузер ВСЕГДА проверяет на сервере
			// перед использованием, но пользуется ETag/Last-Modified для 304.
			w.Header().Set("Cache-Control", "no-cache, must-revalidate")
		case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".avif", ".ico", ".svg":
			// Изображения меняются редко — 5 минут жёсткого кеша + revalidate
			w.Header().Set("Cache-Control", "public, max-age=300, must-revalidate")
		case ".woff", ".woff2", ".ttf", ".otf", ".eot":
			// Шрифты меняются практически никогда — 1 час
			w.Header().Set("Cache-Control", "public, max-age=3600, must-revalidate")
		default:
			// Неизвестное расширение — на всякий случай тоже no-cache
			w.Header().Set("Cache-Control", "no-cache, must-revalidate")
		}

		// Vary: Cookie помогает CDN не отдавать одну и ту же
		// сессионно-зависимую страницу разным пользователям.
		// Безопаснее по умолчанию, не вредит при отсутствии CDN.
		w.Header().Add("Vary", "Cookie")

		h.ServeHTTP(w, r)
	})
}

func injectHTML(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		rec := &htmlInjectWriter{
			ResponseWriter: w,
			buf:            &bytes.Buffer{},
		}
		next.ServeHTTP(rec, r)
		rec.flush()
	})
}

// htmlInjectWriter буферизует ответ, чтобы мы могли решить, HTML это или нет,
// и при необходимости вставить htmlInject перед </head>.
type htmlInjectWriter struct {
	http.ResponseWriter
	buf         *bytes.Buffer
	status      int
	wroteHeader bool
	isHTML      bool
	passThrough bool
}

func (w *htmlInjectWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true

	ct := w.ResponseWriter.Header().Get("Content-Type")
	w.isHTML = strings.HasPrefix(strings.ToLower(ct), "text/html")

	if !w.isHTML || status >= 300 {
		w.passThrough = true
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *htmlInjectWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if w.passThrough {
		return w.ResponseWriter.Write(b)
	}
	return w.buf.Write(b)
}

func (w *htmlInjectWriter) flush() {
	if w.passThrough {
		return
	}
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	body := w.buf.Bytes()
	needle := []byte("</head>")
	idx := bytes.Index(body, needle)
	if idx < 0 {
		needleUpper := []byte("</HEAD>")
		idx = bytes.Index(body, needleUpper)
		if idx >= 0 {
			needle = needleUpper
		}
	}

	if idx < 0 {
		w.ResponseWriter.Header().Del("Content-Length")
		w.ResponseWriter.WriteHeader(w.status)
		_, _ = w.ResponseWriter.Write(body)
		return
	}

	var out bytes.Buffer
	out.Grow(len(body) + len(htmlInject))
	out.Write(body[:idx])
	out.WriteString(htmlInject)
	out.Write(body[idx:])

	w.ResponseWriter.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(w.status)
	_, _ = io.Copy(w.ResponseWriter, &out)
}

func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s status=%d duration=%s ip=%s", r.Method, r.URL.Path, rec.status, time.Since(start), clientIP(r))
	})
}

func clientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func newToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func newPublicUserID() (string, error) {
	buf := make([]byte, 5)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "u" + hex.EncodeToString(buf), nil
}

func updateUserProfile(db *sql.DB, userID int64, req profileUpdateRequest) (user, error) {
	firstName := strings.TrimSpace(req.FirstName)
	lastName := strings.TrimSpace(req.LastName)
	fullName := strings.TrimSpace(req.FullName)
	if firstName == "" {
		return user{}, fmt.Errorf("%w: имя обязательно", errValidation)
	}
	if lastName == "" {
		return user{}, fmt.Errorf("%w: фамилия обязательна", errValidation)
	}
	if fullName == "" {
		fullName = strings.TrimSpace(firstName + " " + lastName)
	}

	var updated user
	err := db.QueryRow(`
		UPDATE users
		SET first_name = $2,
		    last_name = $3,
		    full_name = $4,
		    position = NULLIF($5, ''),
		    company_name = NULLIF($6, ''),
		    bio = NULLIF($7, ''),
		    phone = NULLIF($8, ''),
		    location = NULLIF($9, ''),
		    city = NULLIF($10, ''),
		    avatar_url = NULLIF($11, '')
		WHERE id = $1
		RETURNING id, public_id, first_name, last_name, full_name, email,
			COALESCE(position, ''), COALESCE(company_name, ''), COALESCE(bio, ''),
			COALESCE(phone, ''), COALESCE(location, ''), COALESCE(city, ''), COALESCE(avatar_url, ''), COALESCE(handle, '')
	`, userID,
		firstName,
		lastName,
		fullName,
		strings.TrimSpace(req.Position),
		strings.TrimSpace(req.CompanyName),
		strings.TrimSpace(req.Bio),
		strings.TrimSpace(req.Phone),
		strings.TrimSpace(req.Location),
		strings.TrimSpace(req.City),
		strings.TrimSpace(req.AvatarURL),
	).Scan(&updated.ID, &updated.PublicID, &updated.FirstName, &updated.LastName, &updated.FullName, &updated.Email, &updated.Position, &updated.CompanyName, &updated.Bio, &updated.Phone, &updated.Location, &updated.City, &updated.AvatarURL, &updated.Handle)
	if err != nil {
		return user{}, err
	}
	return updated, nil
}

func setUserHandle(db *sql.DB, userID int64, newHandle string) (user, error) {
	h, err := validateHandle(newHandle)
	if err != nil {
		return user{}, err
	}
	var current string
	var changedAt sql.NullTime
	if err := db.QueryRow(`SELECT COALESCE(handle, ''), handle_changed_at FROM users WHERE id = $1`, userID).Scan(&current, &changedAt); err != nil {
		return user{}, err
	}
	if strings.EqualFold(current, h) {
		return user{}, fmt.Errorf("%w: это уже ваш handle", errConflict)
	}
	if changedAt.Valid && time.Since(changedAt.Time) < 30*24*time.Hour {
		return user{}, fmt.Errorf("%w: можно менять не чаще раза в 30 дней", errValidation)
	}
	_, err = db.Exec(`UPDATE users SET handle = $2, handle_changed_at = NOW() WHERE id = $1`, userID, h)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_handle_lower_uniq_idx" {
			return user{}, errHandleTaken
		}
		return user{}, err
	}
	return getUserByID(db, userID)
}

func checkHandleAvailability(db *sql.DB, handle string) (bool, string) {
	h, err := validateHandle(handle)
	if err != nil {
		return false, err.Error()
	}
	var exists int
	if err := db.QueryRow(`SELECT 1 FROM users WHERE LOWER(handle) = LOWER($1) AND is_deleted = FALSE LIMIT 1`, h).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return true, ""
		}
		return false, "ошибка проверки"
	}
	return false, "занят"
}

func searchUsersByHandle(db *sql.DB, prefix string, limit int) ([]friendCandidateDTO, error) {
	if limit <= 0 {
		limit = 8
	}
	if limit > 20 {
		limit = 20
	}
	prefix = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.TrimSpace(strings.TrimPrefix(prefix, "@")))
	rows, err := db.Query(`
		SELECT public_id, COALESCE(full_name,''), COALESCE(handle,'')
		FROM users
		WHERE handle IS NOT NULL AND is_deleted = FALSE AND LOWER(handle) LIKE LOWER($1) || '%' ESCAPE '\'
		ORDER BY LOWER(handle)
		LIMIT $2
	`, prefix, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []friendCandidateDTO
	for rows.Next() {
		var u friendCandidateDTO
		if err := rows.Scan(&u.ID, &u.FullName, &u.Email); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func getUserByHandle(db *sql.DB, handle string) (publicUserProfile, error) {
	var p publicUserProfile
	err := db.QueryRow(`
		SELECT public_id, first_name, last_name, full_name, email, COALESCE(handle, '')
		FROM users
		WHERE LOWER(handle)=LOWER($1) AND is_deleted = FALSE
	`, strings.TrimSpace(strings.TrimPrefix(handle, "@"))).Scan(&p.PublicID, &p.FirstName, &p.LastName, &p.FullName, &p.Email, &p.Handle)
	return p, err
}

func loadUserSettings(db *sql.DB, userID int64) (userSettings, error) {
	_, _ = db.Exec(`INSERT INTO user_settings(user_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID)
	var s userSettings
	err := db.QueryRow(`
		SELECT notif_email, notif_push, notif_friend_requests, notif_chat_messages, notif_mentions,
		       notif_reactions, notif_new_jobs, notif_events, notif_news_digest, notif_platform_updates,
		       privacy_profile_private, privacy_show_email, privacy_show_phone, privacy_discoverable, privacy_show_online, privacy_show_last_seen, privacy_who_can_message,
		       COALESCE(u.theme,'light'), COALESCE(u.layout_mode,'normal'), COALESCE(u.compact_feed,FALSE), COALESCE(u.locale,'ru'), COALESCE(u.timezone,'Europe/Moscow'), COALESCE(u.date_format,'DD.MM.YYYY')
		FROM user_settings us
		JOIN users u ON u.id = us.user_id
		WHERE us.user_id = $1
	`, userID).Scan(
		&s.NotifEmail, &s.NotifPush, &s.NotifFriendRequests, &s.NotifChatMessages, &s.NotifMentions, &s.NotifReactions, &s.NotifNewJobs, &s.NotifEvents, &s.NotifNewsDigest, &s.NotifPlatformUpdates,
		&s.PrivacyProfilePrivate, &s.PrivacyShowEmail, &s.PrivacyShowPhone, &s.PrivacyDiscoverable, &s.PrivacyShowOnline, &s.PrivacyShowLastSeen, &s.PrivacyWhoCanMessage,
		&s.Theme, &s.LayoutMode, &s.CompactFeed, &s.Locale, &s.Timezone, &s.DateFormat,
	)
	return s, err
}

// computePlatformStats собирает агрегированные данные платформы.
// Использует только COUNT-запросы и индексы. Тяжёлых JOIN'ов нет.
func computePlatformStats(db *sql.DB) (map[string]any, error) {
	out := map[string]any{
		"members":       0,
		"news":          0,
		"events":        0,
		"active_now":    0,
		"top_news_week": nil,
		"trend_today":   nil,
	}

	// 1. Members
	var members int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE is_deleted = FALSE`).Scan(&members); err != nil {
		return nil, err
	}
	out["members"] = members

	// 2. News (опубликованные посты)
	var news int64
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM posts
		WHERE type = 'news' AND is_deleted = FALSE AND privacy_level = 'public'
	`).Scan(&news); err != nil {
		return nil, err
	}
	out["news"] = news

	// 3. Events (актуальные/будущие)
	var events int64
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM events
		WHERE is_deleted = FALSE
		  AND status = 'published'
		  AND (ends_at >= NOW() OR starts_at >= NOW())
	`).Scan(&events); err != nil {
		return nil, err
	}
	out["events"] = events

	// 4. Active now — уникальные user_id с активной сессией за последние 5 минут
	var activeNow int64
	if err := db.QueryRow(`
		SELECT COUNT(DISTINCT user_id) FROM sessions
		WHERE last_seen_at > NOW() - INTERVAL '5 minutes'
		  AND expires_at > NOW()
	`).Scan(&activeNow); err != nil {
		return nil, err
	}
	out["active_now"] = activeNow

	// 5. Top news week — посты за 7 дней по weighted score.
	var topPublicID, topTitle string
	err := db.QueryRow(`
		SELECT public_id, title
		FROM posts
		WHERE type = 'news'
		  AND is_deleted = FALSE
		  AND privacy_level = 'public'
		  AND created_at > NOW() - INTERVAL '7 days'
		ORDER BY (likes_count * 3 + comments_count * 2 + saves_count * 5 + views_count * 0.1) DESC,
		         created_at DESC
		LIMIT 1
	`).Scan(&topPublicID, &topTitle)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	} else if topPublicID != "" {
		out["top_news_week"] = map[string]any{
			"public_id": topPublicID,
			"title":     topTitle,
		}
	}

	// 6. Trend today — самый частый тег в постах за последние 24 часа.
	var trendTag string
	err = db.QueryRow(`
		SELECT tag FROM (
			SELECT UNNEST(tags) AS tag
			FROM posts
			WHERE created_at > NOW() - INTERVAL '24 hours'
			  AND type = 'news'
			  AND is_deleted = FALSE
			  AND privacy_level = 'public'
			  AND tags IS NOT NULL
		) AS t
		WHERE tag IS NOT NULL AND tag <> ''
		GROUP BY tag
		ORDER BY COUNT(*) DESC, tag ASC
		LIMIT 1
	`).Scan(&trendTag)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	} else if trendTag != "" {
		out["trend_today"] = trendTag
	}

	return out, nil
}

func updateNotificationSettings(db *sql.DB, userID int64, req updateNotificationsRequest) error {
	_, err := db.Exec(`INSERT INTO user_settings(user_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE user_settings SET
		notif_email=COALESCE($2,notif_email), notif_push=COALESCE($3,notif_push), notif_friend_requests=COALESCE($4,notif_friend_requests),
		notif_chat_messages=COALESCE($5,notif_chat_messages), notif_mentions=COALESCE($6,notif_mentions), notif_reactions=COALESCE($7,notif_reactions),
		notif_new_jobs=COALESCE($8,notif_new_jobs), notif_events=COALESCE($9,notif_events), notif_news_digest=COALESCE($10,notif_news_digest),
		notif_platform_updates=COALESCE($11,notif_platform_updates), updated_at=NOW()
		WHERE user_id=$1`,
		userID, req.NotifEmail, req.NotifPush, req.NotifFriendRequests, req.NotifChatMessages, req.NotifMentions, req.NotifReactions, req.NotifNewJobs, req.NotifEvents, req.NotifNewsDigest, req.NotifPlatformUpdates)
	return err
}

func updatePrivacySettings(db *sql.DB, userID int64, req updatePrivacyRequest) error {
	if req.PrivacyWhoCanMessage != nil {
		v := strings.TrimSpace(*req.PrivacyWhoCanMessage)
		if v != "all" && v != "contacts" && v != "nobody" {
			return fmt.Errorf("%w: privacy_who_can_message должен быть all/contacts/nobody", errValidation)
		}
		req.PrivacyWhoCanMessage = &v
	}
	_, _ = db.Exec(`INSERT INTO user_settings(user_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID)
	_, err := db.Exec(`UPDATE user_settings SET
		privacy_profile_private=COALESCE($2,privacy_profile_private), privacy_show_email=COALESCE($3,privacy_show_email), privacy_show_phone=COALESCE($4,privacy_show_phone),
		privacy_discoverable=COALESCE($5,privacy_discoverable), privacy_show_online=COALESCE($6,privacy_show_online), privacy_show_last_seen=COALESCE($7,privacy_show_last_seen),
		privacy_who_can_message=COALESCE($8,privacy_who_can_message), updated_at=NOW()
		WHERE user_id=$1`, userID, req.PrivacyProfilePrivate, req.PrivacyShowEmail, req.PrivacyShowPhone, req.PrivacyDiscoverable, req.PrivacyShowOnline, req.PrivacyShowLastSeen, req.PrivacyWhoCanMessage)
	return err
}

func updateAppearanceSettings(db *sql.DB, userID int64, req updateAppearanceRequest) error {
	if req.Theme != nil {
		v := strings.TrimSpace(*req.Theme)
		if v != "light" && v != "dark" && v != "auto" {
			return fmt.Errorf("%w: theme должен быть light/dark/auto", errValidation)
		}
		req.Theme = &v
	}
	if req.LayoutMode != nil {
		v := strings.TrimSpace(*req.LayoutMode)
		if v != "normal" && v != "wide" {
			return fmt.Errorf("%w: layout_mode должен быть normal/wide", errValidation)
		}
		req.LayoutMode = &v
	}
	if req.Locale != nil {
		v := strings.TrimSpace(*req.Locale)
		if v != "ru" && v != "en" {
			return fmt.Errorf("%w: locale должен быть ru/en", errValidation)
		}
		req.Locale = &v
	}
	if req.DateFormat != nil {
		v := strings.TrimSpace(*req.DateFormat)
		if v != "DD.MM.YYYY" && v != "YYYY-MM-DD" && v != "MM/DD/YYYY" {
			return fmt.Errorf("%w: date_format некорректен", errValidation)
		}
		req.DateFormat = &v
	}
	if req.Timezone != nil {
		v := strings.TrimSpace(*req.Timezone)
		if len(v) > 64 || v == "" {
			return fmt.Errorf("%w: timezone некорректен", errValidation)
		}
		req.Timezone = &v
	}
	_, err := db.Exec(`UPDATE users SET
		theme=COALESCE($2,theme), layout_mode=COALESCE($3,layout_mode), compact_feed=COALESCE($4,compact_feed),
		locale=COALESCE($5,locale), timezone=COALESCE($6,timezone), date_format=COALESCE($7,date_format)
		WHERE id=$1`, userID, req.Theme, req.LayoutMode, req.CompactFeed, req.Locale, req.Timezone, req.DateFormat)
	return err
}

func changeUserEmail(db *sql.DB, userID int64, newEmail, currentPassword string) error {
	email := strings.ToLower(strings.TrimSpace(newEmail))
	if !isValidEmail(email) {
		return fmt.Errorf("%w: email некорректен", errValidation)
	}
	var hash string
	if err := db.QueryRow(`SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&hash); err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(currentPassword)) != nil {
		return fmt.Errorf("%w: неверный пароль", errValidation)
	}
	_, err := db.Exec(`UPDATE users SET email = $2 WHERE id = $1`, userID, email)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return errConflict
		}
	}
	return err
}

func changeUserPassword(db *sql.DB, sessions *sessionStore, userID int64, currentPassword, newPassword, exceptToken string) error {
	if len(newPassword) < 8 {
		return fmt.Errorf("%w: новый пароль должен быть не короче 8 символов", errValidation)
	}
	var hash string
	if err := db.QueryRow(`SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&hash); err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(currentPassword)) != nil {
		return fmt.Errorf("%w: неверный пароль", errValidation)
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(newPassword)) == nil {
		return fmt.Errorf("%w: новый пароль должен отличаться от текущего", errValidation)
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE users SET password_hash = $2, last_password_change_at = NOW() WHERE id = $1`, userID, string(newHash)); err != nil {
		return err
	}
	sessions.invalidateUserExcept(userID, exceptToken)
	return nil
}

func deleteUserAccount(db *sql.DB, sessions *sessionStore, userID int64, currentPassword, confirmation string) error {
	if confirmation != "УДАЛИТЬ" {
		return fmt.Errorf("%w: подтверждение должно быть словом УДАЛИТЬ", errValidation)
	}
	var hash string
	if err := db.QueryRow(`SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&hash); err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(currentPassword)) != nil {
		return fmt.Errorf("%w: неверный пароль", errValidation)
	}
	_, err := db.Exec(`
		UPDATE users
		SET is_deleted = TRUE, deleted_at = NOW(), handle = NULL,
		    email = CONCAT('deleted-', id, '@deleted.local'),
		    full_name = 'Удалённый пользователь', first_name = 'Удалённый', last_name = '',
		    position = NULL, company_name = NULL, bio = NULL, phone = NULL, location = NULL, city = NULL, avatar_url = NULL
		WHERE id = $1
	`, userID)
	if err != nil {
		return err
	}
	sessions.invalidateUser(userID)
	return nil
}

func parseMentionsFromText(text string) []string {
	m := mentionRe.FindAllStringSubmatch(text, -1)
	uniq := map[string]struct{}{}
	out := make([]string, 0, len(m))
	for _, hit := range m {
		if len(hit) < 2 {
			continue
		}
		h := strings.ToLower(hit[1])
		if _, ok := uniq[h]; ok {
			continue
		}
		uniq[h] = struct{}{}
		out = append(out, h)
	}
	return out
}

func saveMentions(db *sql.DB, sourceType string, sourceID, actorUserID int64, text, preview string) error {
	handles := parseMentionsFromText(text)
	if len(handles) == 0 {
		return nil
	}
	rows, err := db.Query(`SELECT id FROM users WHERE LOWER(handle) = ANY($1) AND is_deleted = FALSE`, pgtype.FlatArray[string](handles))
	if err != nil {
		return err
	}
	defer rows.Close()
	actorName := getUserDisplayName(db, actorUserID)
	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err != nil {
			return err
		}
		if uid == actorUserID {
			continue
		}
		_, _ = db.Exec(`INSERT INTO user_mentions(mentioned_user_id, actor_user_id, source_type, source_id, preview) VALUES ($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING`,
			uid, actorUserID, sourceType, sourceID, preview)

		var sourcePublicID string
		if sourceType == "post" {
			_ = db.QueryRow(`SELECT public_id FROM posts WHERE id = $1`, sourceID).Scan(&sourcePublicID)
		} else if sourceType == "comment" {
			_ = db.QueryRow(`SELECT p.public_id FROM posts p JOIN post_comments pc ON pc.post_id = p.id WHERE pc.id = $1`, sourceID).Scan(&sourcePublicID)
		}
		title := actorName + " упомянул вас"
		if err := createNotification(db, createNotificationParams{
			RecipientID:    uid,
			ActorID:        actorUserID,
			Type:           "mention",
			SourceType:     sourceType,
			SourceID:       sourceID,
			SourcePublicID: sourcePublicID,
			Title:          title,
			Preview:        preview,
		}); err != nil {
			log.Printf("notif mention: %v", err)
		}
	}
	return rows.Err()
}
