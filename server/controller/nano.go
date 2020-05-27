package controller

import (
	"fmt"
	"math"
	"time"

	"github.com/appditto/natricon/server/db"
	"github.com/appditto/natricon/server/image"
	"github.com/appditto/natricon/server/net"
	"github.com/appditto/natricon/server/utils"
	"github.com/bsm/redislock"
	"github.com/golang/glog"
	socketio "github.com/googollee/go-socket.io"
)

// Donations at or above this threshold will award "vip" status for 30 days
const donationThresholdNano = 2.0

type NanoController struct {
	RPCClient       *net.RPCClient
	SIOServer       *socketio.Server
	DonationAccount string
}

// Handle callback for donation listener
func (nc NanoController) Callback(confirmationResponse net.ConfirmationResponse) {
	block := confirmationResponse.Message["block"].(map[string]interface{})
	amount := confirmationResponse.Message["amount"].(string)
	hash := confirmationResponse.Message["hash"].(string)
	// Check if send to donation account
	if block["link_as_account"] == nc.DonationAccount && block["link_as_account"] != block["account"] {
		// Emit SIO event
		data := map[string]string{
			"amount": amount,
		}
		nc.SIOServer.BroadcastToRoom("", "bcast", "donation_event", data)
		// Calc donor duration with lock
		lock, err := db.GetDB().Locker.Obtain(fmt.Sprintf("natricon:callback_lock:%s", hash), 100*time.Second, nil)
		if err == redislock.ErrNotObtained {
			return
		} else if err != nil {
			glog.Error(err)
			return
		}
		defer lock.Release()
		durationDays := nc.calcDonorDurationDays(amount)
		if durationDays > 0 {
			glog.Infof("Giving donor status to %s for %d days", block["account"], durationDays)
			db.GetDB().UpdateDonorStatus(hash, block["account"].(string), durationDays)
		}
		// Issue refund for odd raw amounts
		asNano, err := utils.RawToNano(amount, false)
		if err != nil {
			return
		}
		asNanoTrunc := math.Trunc(asNano)
		asNano -= asNanoTrunc
		if len(amount) >= 28 && asNano > 0.001 {
			// Refund
			refundRaw := amount[len(amount)-28:]
			refundNano, err := utils.RawToNano(refundRaw, false)
			if err != nil {
				return
			}
			// If refund is 0, don't do it
			if refundNano == 0 {
				return
			} else if refundRaw == amount {
				// If refund is equal to the total amount don't send refund
				return
			} else if len(refundRaw) > 28 {
				// In case something goes wrong with previous code
				glog.Errorf("Attempted to refund an amount that was larger than expected %s", refundRaw)
				return
			}
			glog.Infof("Going to refund %s raw to %s", refundRaw, block["account"])
			// Send refund
			wallet := utils.GetEnv("WALLET_ID", "")
			if wallet == "" {
				glog.Warningf("Not issuing refund for %s because WALLET_ID is not configured", hash)
				return
			}
			response, err := nc.RPCClient.MakeSendRequest(
				nc.DonationAccount,
				block["account"].(string),
				refundRaw,
				hash,
				wallet,
			)
			if err != nil {
				glog.Errorf("Failed to issue refund of %s to %s for %s", refundRaw, block["account"].(string), hash)
				return
			}
			glog.Infof("Issued refund with hash %s", response.Block)
		}
	}
}

// Cron job for checking missed callbacks
func (nc NanoController) CheckMissedCallbacks() {
	if nc.RPCClient == nil {
		return
	}

	// Try to obtain lock.
	lock, err := db.GetDB().Locker.Obtain("natricon:history_lock", 100*time.Second, nil)
	if err == redislock.ErrNotObtained {
		return
	} else if err != nil {
		glog.Error(err)
		return
	}
	defer lock.Release()
	// Check history
	historyResponse, err := nc.RPCClient.MakeAccountHistoryRequest(
		nc.DonationAccount,
		10,
	)
	if err != nil {
		glog.Errorf("Error occured checking donation account history %s", err)
		return
	}
	for i := 0; i < len(historyResponse.History); i++ {
		if historyResponse.History[i].Type == "receive" && historyResponse.History[i].Account != nc.DonationAccount {
			durationDays := nc.calcDonorDurationDays(historyResponse.History[i].Amount)
			if durationDays > 0 {
				glog.Infof("Checking donor status to %s for %d days", historyResponse.History[i].Account, durationDays)
				db.GetDB().UpdateDonorStatus(historyResponse.History[i].Hash, historyResponse.History[i].Account, durationDays)
			}
		}
	}
}

// calcDonorDurationDays - calculate how long badge will persist with given donation amount
func (nc NanoController) calcDonorDurationDays(amountRaw string) uint {
	amountNano, _ := utils.RawToNano(amountRaw, true)
	// TODO - allow partial chunks?
	chunks := uint(amountNano / donationThresholdNano)
	return chunks * 30
}

// Cron job for updating principal rep weight requirement
func (nc NanoController) UpdatePrincipalWeight() {
	if nc.RPCClient == nil {
		return
	}
	// Check history
	quorumResponse, err := nc.RPCClient.MakeConfirmationQuorumRequest()
	if err != nil {
		glog.Errorf("Error occured checking confirmation quorum %s", err)
		return
	}
	onlineWeightMinimum, err := utils.RawToNano(quorumResponse.OnlineWeightTotal, true)
	if err != nil {
		glog.Errorf("Error occured converting weight to nano %s", err)
		return
	}
	// 0.1% of online weight means principal rep
	principalRepMinimum := onlineWeightMinimum * 0.001
	glog.Infof("Setting principal rep requirement to %f", principalRepMinimum)
	db.GetDB().SetPrincipalRepRequirement(principalRepMinimum)
}

// Cron job for updating principal reps
func (nc NanoController) UpdatePrincipalReps() {
	if nc.RPCClient == nil {
		return
	}
	glog.Infof("Updating principal rep list")
	// Get weight requirement
	repWeightRequirement := db.GetDB().GetPrincipalRepRequirement()
	// Get reps
	repsResponse, err := nc.RPCClient.MakeRepresentativesRequest()
	if err != nil {
		glog.Errorf("Error occured checking confirmation quorum %s", err)
		return
	}
	principalReps := []string{}
	for rep, weight := range repsResponse.Representatives {
		weightNano, err := utils.RawToNano(weight, true)
		if err != nil {
			glog.Errorf("Error occured checking weight for rep %s %s", rep, err)
			continue
		}
		if weightNano >= repWeightRequirement {
			principalReps = append(principalReps, utils.AddressToPub(rep))
		}
	}
	// Update cache
	db.GetDB().SetPrincipalReps(principalReps)
	// Update badge service
	image.GetBadgeSvc().UpdatePrincipalReps(principalReps)
}
