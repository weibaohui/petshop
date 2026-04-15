// Package handlers provides HTTP handlers for the petshop API.
package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"petshop/internal/cache"
	"petshop/internal/logger"
	"petshop/internal/models"
)

var (
	votes      []models.Vote
	votesMu    sync.RWMutex
	allStars   []models.PetAllStar
	allStarsMu sync.RWMutex
	voteLogger = logger.New("votes")
)

func init() {
	votes = []models.Vote{}
	allStars = []models.PetAllStar{}
}

func VoteForPet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	var req struct {
		PetID  int64 `json:"petId"`
		UserID int64 `json:"userId"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON format"})
		return
	}

	if req.PetID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "petId is required"})
		return
	}
	if req.UserID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "userId is required"})
		return
	}

	petsMu.RLock()
	var petFound bool
	for _, pet := range pets {
		if pet.ID == req.PetID {
			petFound = true
			break
		}
	}
	petsMu.RUnlock()

	if !petFound {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "pet not found"})
		return
	}

	votesMu.Lock()
	defer votesMu.Unlock()

	for _, v := range votes {
		if v.PetID == req.PetID && v.UserID == req.UserID {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "you have already voted for this pet"})
			return
		}
	}

	newVote := models.Vote{
		ID:        int64(len(votes) + 1),
		PetID:     req.PetID,
		UserID:    req.UserID,
		CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
	}
	votes = append(votes, newVote)

	petsMu.Lock()
	for i, pet := range pets {
		if pet.ID == req.PetID {
			pets[i].VoteCount++
			petCache.Delete(cache.GetPetKey(pet.ID))
			break
		}
	}
	petsMu.Unlock()

	voteLogger.Info("vote recorded", map[string]interface{}{
		"petId":  req.PetID,
		"userId": req.UserID,
	})

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   "vote recorded successfully",
		"voteCount": getVoteCountForPet(req.PetID),
		"hasVoted":  true,
	})
}

func getVoteCountForPet(petID int64) int {
	votesMu.RLock()
	defer votesMu.RUnlock()
	count := 0
	for _, v := range votes {
		if v.PetID == petID {
			count++
		}
	}
	return count
}

func GetVoteStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	petIDStr := r.URL.Query().Get("petId")
	userIDStr := r.URL.Query().Get("userId")

	if petIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "petId is required"})
		return
	}

	petID, err := strconv.ParseInt(petIDStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid petId format"})
		return
	}

	votesMu.RLock()
	defer votesMu.RUnlock()

	hasVoted := false
	if userIDStr != "" {
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err == nil {
			for _, v := range votes {
				if v.PetID == petID && v.UserID == userID {
					hasVoted = true
					break
				}
			}
		}
	}

	voteCount := 0
	for _, v := range votes {
		if v.PetID == petID {
			voteCount++
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"petId":     petID,
		"voteCount": voteCount,
		"hasVoted":  hasVoted,
	})
}

func GetPetVotes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	petIDStr := r.URL.Query().Get("petId")
	if petIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "petId is required"})
		return
	}

	petID, err := strconv.ParseInt(petIDStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid petId format"})
		return
	}

	votesMu.RLock()
	defer votesMu.RUnlock()

	var petVotes []models.Vote
	for _, v := range votes {
		if v.PetID == petID {
			petVotes = append(petVotes, v)
		}
	}

	json.NewEncoder(w).Encode(petVotes)
}

func ElectAllStar(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	petsMu.RLock()
	var maxVotes int
	var topPetID int64
	for _, pet := range pets {
		if pet.VoteCount > maxVotes {
			maxVotes = pet.VoteCount
			topPetID = pet.ID
		}
	}
	petsMu.RUnlock()

	if topPetID == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "no pets available for election"})
		return
	}

	petsMu.RLock()
	var topPet *models.Pet
	for _, pet := range pets {
		if pet.ID == topPetID {
			topPet = &pet
			break
		}
	}
	petsMu.RUnlock()

	if topPet == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "pet not found"})
		return
	}

	now := time.Now()
	period := fmt.Sprintf("%d-%02d", now.Year(), now.Month())

	allStarsMu.Lock()
	defer allStarsMu.Unlock()

	newStar := models.PetAllStar{
		ID:        int64(len(allStars) + 1),
		PetID:     topPetID,
		Pet:       topPet,
		VoteCount: maxVotes,
		ElectedAt: now.Format("2006-01-02 15:04:05"),
		Period:    period,
	}
	allStars = append(allStars, newStar)

	voteLogger.Info("all-star elected", map[string]interface{}{
		"petId":     topPetID,
		"petName":   topPet.Name,
		"voteCount": maxVotes,
		"period":    period,
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(newStar)
}

func GetCurrentAllStar(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	allStarsMu.RLock()
	defer allStarsMu.RUnlock()

	if len(allStars) == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "no all-star elected yet"})
		return
	}

	currentStar := allStars[len(allStars)-1]
	json.NewEncoder(w).Encode(currentStar)
}

func GetAllStars(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	allStarsMu.RLock()
	defer allStarsMu.RUnlock()

	if len(allStars) == 0 {
		json.NewEncoder(w).Encode([]models.PetAllStar{})
		return
	}

	json.NewEncoder(w).Encode(allStars)
}

func GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	petsMu.RLock()
	defer petsMu.RUnlock()

	type petWithVotes struct {
		models.Pet
		VoteCount int `json:"voteCount"`
	}

	var leaderboard []petWithVotes
	for _, pet := range pets {
		leaderboard = append(leaderboard, petWithVotes{
			Pet:       pet,
			VoteCount: getVoteCountForPet(pet.ID),
		})
	}

	for i := 0; i < len(leaderboard)-1; i++ {
		for j := i + 1; j < len(leaderboard); j++ {
			if leaderboard[j].VoteCount > leaderboard[i].VoteCount {
				leaderboard[i], leaderboard[j] = leaderboard[j], leaderboard[i]
			}
		}
	}

	json.NewEncoder(w).Encode(leaderboard)
}
