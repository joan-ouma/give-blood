package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/joan-ouma/give-blood/internal/config"
	"github.com/joan-ouma/give-blood/internal/db"
	"github.com/joan-ouma/give-blood/internal/entities"
	"github.com/joan-ouma/give-blood/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	database, err := db.Connect(cfg.MongoURI)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Close()

	mongoDB := database.Client.Database("blood_donation")
	ctx := context.Background()

	// Check if database has existing users
	userCount, err := mongoDB.Collection("users").CountDocuments(ctx, bson.M{})
	if err == nil && userCount > 0 && os.Getenv("FORCE_SEED") != "true" {
		log.Println("Database already contains users. Skipping seed to protect existing data.")
		log.Println("To force seeding and drop all collections, run with FORCE_SEED=true environment variable.")
		return
	}

	// Clear existing collections
	log.Println("Clearing existing collections...")
	_ = mongoDB.Collection("users").Drop(ctx)
	_ = mongoDB.Collection("locations").Drop(ctx)
	_ = mongoDB.Collection("drives").Drop(ctx)
	_ = mongoDB.Collection("donations").Drop(ctx)
	_ = mongoDB.Collection("donor_stats").Drop(ctx)

	// Ensure Indexes
	log.Println("Ensuring indexes...")
	userService := service.NewUserService(mongoDB)
	locationService := service.NewLocationService(mongoDB)
	driveService := service.NewDriveService(mongoDB)
	donationService := service.NewDonationService(mongoDB)
	leaderboardService := service.NewLeaderboardService(mongoDB)

	_ = userService.EnsureIndexes(ctx)
	_ = locationService.EnsureIndexes(ctx)
	_ = driveService.EnsureIndexes(ctx)
	_ = donationService.EnsureIndexes(ctx)
	_ = leaderboardService.EnsureIndexes(ctx)

	// Setup password hash
	passHash, err := service.HashPassword("password123")
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}

	joanoumaPassHash, err := service.HashPassword("assembly1234")
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}

	latNairobi := -1.2921
	lngNairobi := 36.8219
	latMombasa := -4.0435
	lngMombasa := 39.6682

	// 1. Create Demo Agencies
	log.Println("Creating agencies...")
	agency1 := entities.User{
		ID:           primitive.NewObjectID(),
		Email:        "agency@redcross.org",
		PasswordHash: passHash,
		Role:         entities.RoleAgency,
		Name:         "Red Cross Nairobi",
		Lat:          &latNairobi,
		Lng:          &lngNairobi,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	agency2 := entities.User{
		ID:           primitive.NewObjectID(),
		Email:        "agency@mombasabb.org",
		PasswordHash: passHash,
		Role:         entities.RoleAgency,
		Name:         "Mombasa Blood Bank",
		Lat:          &latMombasa,
		Lng:          &lngMombasa,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	_, _ = mongoDB.Collection("users").InsertMany(ctx, []interface{}{agency1, agency2})

	// 2. Create Locations
	log.Println("Creating locations...")

	loc1 := entities.Location{
		ID:        primitive.NewObjectID(),
		AgencyID:  agency1.ID,
		Name:      "Nairobi HQ Clinic",
		Address:   "123 Red Cross Road",
		City:      "Nairobi",
		Lat:       &latNairobi,
		Lng:       &lngNairobi,
		Hours:     "08:00 - 17:00",
		Phone:     "+254 700 000 001",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	loc2 := entities.Location{
		ID:        primitive.NewObjectID(),
		AgencyID:  agency1.ID,
		Name:      "Westlands Donation Center",
		Address:   "Westlands Mall Ground Floor",
		City:      "Nairobi",
		Lat:       &latNairobi,
		Lng:       &lngNairobi,
		Hours:     "09:00 - 19:00",
		Phone:     "+254 700 000 002",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	loc3 := entities.Location{
		ID:        primitive.NewObjectID(),
		AgencyID:  agency2.ID,
		Name:      "Mombasa Island Center",
		Address:   "456 Digo Road",
		City:      "Mombasa",
		Lat:       &latMombasa,
		Lng:       &lngMombasa,
		Hours:     "08:00 - 17:00",
		Phone:     "+254 700 000 003",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_, _ = mongoDB.Collection("locations").InsertMany(ctx, []interface{}{loc1, loc2, loc3})

	// 3. Create Drives
	log.Println("Creating drives...")
	drive1 := entities.Drive{
		ID:         primitive.NewObjectID(),
		AgencyID:   agency1.ID,
		LocationID: &loc1.ID,
		Title:      "Nairobi Plaza Drive",
		Address:    "Nairobi Plaza Courtyard",
		City:       "Nairobi",
		StartsAt:   time.Now().UTC().Add(24 * time.Hour),
		EndsAt:     time.Now().UTC().Add(30 * time.Hour),
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	drive2 := entities.Drive{
		ID:         primitive.NewObjectID(),
		AgencyID:   agency1.ID,
		LocationID: &loc2.ID,
		Title:      "Kenyatta University Drive",
		Address:    "Student Center Hall A",
		City:       "Nairobi",
		StartsAt:   time.Now().UTC().Add(48 * time.Hour),
		EndsAt:     time.Now().UTC().Add(54 * time.Hour),
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	drive3 := entities.Drive{
		ID:         primitive.NewObjectID(),
		AgencyID:   agency2.ID,
		LocationID: &loc3.ID,
		Title:      "Mombasa Town Hall Drive",
		Address:    "Town Hall Assembly Square",
		City:       "Mombasa",
		StartsAt:   time.Now().UTC().Add(72 * time.Hour),
		EndsAt:     time.Now().UTC().Add(78 * time.Hour),
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	_, _ = mongoDB.Collection("drives").InsertMany(ctx, []interface{}{drive1, drive2, drive3})

	// 4. Create Donors
	log.Println("Creating donors...")
	latAlice := -1.3000
	lngAlice := 36.8000
	latBob := -1.3100
	lngBob := 36.8100
	latCharlie := -1.3200
	lngCharlie := 36.8200
	latDavid := -1.3300
	lngDavid := 36.8300
	latJoan := -1.2950
	lngJoan := 36.8250

	donorAlice := entities.User{
		ID:           primitive.NewObjectID(),
		Email:        "alice@donor.com",
		PasswordHash: passHash,
		Role:         entities.RoleDonor,
		Name:         "Alice Johnson",
		Lat:          &latAlice,
		Lng:          &lngAlice,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	donorBob := entities.User{
		ID:           primitive.NewObjectID(),
		Email:        "bob@donor.com",
		PasswordHash: passHash,
		Role:         entities.RoleDonor,
		Name:         "Bob Smith",
		Lat:          &latBob,
		Lng:          &lngBob,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	donorCharlie := entities.User{
		ID:           primitive.NewObjectID(),
		Email:        "charlie@donor.com",
		PasswordHash: passHash,
		Role:         entities.RoleDonor,
		Name:         "Charlie Green",
		Lat:          &latCharlie,
		Lng:          &lngCharlie,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	donorDavid := entities.User{
		ID:           primitive.NewObjectID(),
		Email:        "david@donor.com",
		PasswordHash: passHash,
		Role:         entities.RoleDonor,
		Name:         "David Miller",
		Lat:          &latDavid,
		Lng:          &lngDavid,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	donorJoan := entities.User{
		ID:           primitive.NewObjectID(),
		Email:        "joanouma@gmail.com",
		PasswordHash: joanoumaPassHash,
		Role:         entities.RoleDonor,
		Name:         "joanouma",
		Lat:          &latJoan,
		Lng:          &lngJoan,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	_, _ = mongoDB.Collection("users").InsertMany(ctx, []interface{}{donorAlice, donorBob, donorCharlie, donorDavid, donorJoan})

	// 5. Create Donations and Leaderboard entries (donor_stats)
	log.Println("Creating donations and stats...")

	now := time.Now().UTC()
	aliceDonationDate := now.Add(-30 * 24 * time.Hour)
	aliceNextEligible := aliceDonationDate.Add(56 * 24 * time.Hour)

	donationAlice1 := entities.Donation{
		ID:             primitive.NewObjectID(),
		DonorID:        donorAlice.ID,
		AgencyID:       agency1.ID,
		LocationID:     &loc1.ID,
		Pints:          2,
		Status:         entities.StatusVerified,
		DonatedAt:      aliceDonationDate,
		VerifiedAt:     &aliceDonationDate,
		VerifiedBy:     &agency1.ID,
		NextEligibleAt: &aliceNextEligible,
		CreatedAt:      aliceDonationDate,
		UpdatedAt:      aliceDonationDate,
	}

	aliceDonationDate2 := now.Add(-90 * 24 * time.Hour)
	aliceNextEligible2 := aliceDonationDate2.Add(56 * 24 * time.Hour)
	donationAlice2 := entities.Donation{
		ID:             primitive.NewObjectID(),
		DonorID:        donorAlice.ID,
		AgencyID:       agency1.ID,
		LocationID:     &loc1.ID,
		Pints:          2,
		Status:         entities.StatusVerified,
		DonatedAt:      aliceDonationDate2,
		VerifiedAt:     &aliceDonationDate2,
		VerifiedBy:     &agency1.ID,
		NextEligibleAt: &aliceNextEligible2,
		CreatedAt:      aliceDonationDate2,
		UpdatedAt:      aliceDonationDate2,
	}

	aliceDonationDate3 := now.Add(-150 * 24 * time.Hour)
	aliceNextEligible3 := aliceDonationDate3.Add(56 * 24 * time.Hour)
	donationAlice3 := entities.Donation{
		ID:             primitive.NewObjectID(),
		DonorID:        donorAlice.ID,
		AgencyID:       agency1.ID,
		LocationID:     &loc1.ID,
		Pints:          2,
		Status:         entities.StatusVerified,
		DonatedAt:      aliceDonationDate3,
		VerifiedAt:     &aliceDonationDate3,
		VerifiedBy:     &agency1.ID,
		NextEligibleAt: &aliceNextEligible3,
		CreatedAt:      aliceDonationDate3,
		UpdatedAt:      aliceDonationDate3,
	}

	aliceDonationDate4 := now.Add(-210 * 24 * time.Hour)
	aliceNextEligible4 := aliceDonationDate4.Add(56 * 24 * time.Hour)
	donationAlice4 := entities.Donation{
		ID:             primitive.NewObjectID(),
		DonorID:        donorAlice.ID,
		AgencyID:       agency1.ID,
		LocationID:     &loc1.ID,
		Pints:          1,
		Status:         entities.StatusVerified,
		DonatedAt:      aliceDonationDate4,
		VerifiedAt:     &aliceDonationDate4,
		VerifiedBy:     &agency1.ID,
		NextEligibleAt: &aliceNextEligible4,
		CreatedAt:      aliceDonationDate4,
		UpdatedAt:      aliceDonationDate4,
	}

	aliceDonationDate5 := now.Add(-270 * 24 * time.Hour)
	aliceNextEligible5 := aliceDonationDate5.Add(56 * 24 * time.Hour)
	donationAlice5 := entities.Donation{
		ID:             primitive.NewObjectID(),
		DonorID:        donorAlice.ID,
		AgencyID:       agency1.ID,
		LocationID:     &loc1.ID,
		Pints:          1,
		Status:         entities.StatusVerified,
		DonatedAt:      aliceDonationDate5,
		VerifiedAt:     &aliceDonationDate5,
		VerifiedBy:     &agency1.ID,
		NextEligibleAt: &aliceNextEligible5,
		CreatedAt:      aliceDonationDate5,
		UpdatedAt:      aliceDonationDate5,
	}

	bobDonationDate1 := now.Add(-60 * 24 * time.Hour)
	bobNextEligible1 := bobDonationDate1.Add(56 * 24 * time.Hour)
	donationBob1 := entities.Donation{
		ID:             primitive.NewObjectID(),
		DonorID:        donorBob.ID,
		AgencyID:       agency1.ID,
		LocationID:     &loc1.ID,
		Pints:          2,
		Status:         entities.StatusVerified,
		DonatedAt:      bobDonationDate1,
		VerifiedAt:     &bobDonationDate1,
		VerifiedBy:     &agency1.ID,
		NextEligibleAt: &bobNextEligible1,
		CreatedAt:      bobDonationDate1,
		UpdatedAt:      bobDonationDate1,
	}

	bobDonationDate2 := now.Add(-120 * 24 * time.Hour)
	bobNextEligible2 := bobDonationDate2.Add(56 * 24 * time.Hour)
	donationBob2 := entities.Donation{
		ID:             primitive.NewObjectID(),
		DonorID:        donorBob.ID,
		AgencyID:       agency1.ID,
		LocationID:     &loc1.ID,
		Pints:          2,
		Status:         entities.StatusVerified,
		DonatedAt:      bobDonationDate2,
		VerifiedAt:     &bobDonationDate2,
		VerifiedBy:     &agency1.ID,
		NextEligibleAt: &bobNextEligible2,
		CreatedAt:      bobDonationDate2,
		UpdatedAt:      bobDonationDate2,
	}

	donationBobPending := entities.Donation{
		ID:         primitive.NewObjectID(),
		DonorID:    donorBob.ID,
		AgencyID:   agency1.ID,
		LocationID: &loc1.ID,
		Pints:      1,
		Status:     entities.StatusPending,
		DonatedAt:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	rejReason := "Low iron levels"
	donationBobRejected := entities.Donation{
		ID:              primitive.NewObjectID(),
		DonorID:         donorBob.ID,
		AgencyID:        agency1.ID,
		LocationID:      &loc1.ID,
		Pints:           1,
		Status:          entities.StatusRejected,
		DonatedAt:       now.Add(-7 * 24 * time.Hour),
		RejectionReason: &rejReason,
		CreatedAt:       now.Add(-7 * 24 * time.Hour),
		UpdatedAt:       now.Add(-7 * 24 * time.Hour),
	}

	charlieDonationDate := now.Add(-60 * 24 * time.Hour)
	charlieNextEligible := charlieDonationDate.Add(56 * 24 * time.Hour)
	donationCharlie1 := entities.Donation{
		ID:             primitive.NewObjectID(),
		DonorID:        donorCharlie.ID,
		AgencyID:       agency2.ID,
		LocationID:     &loc3.ID,
		Pints:          1,
		Status:         entities.StatusVerified,
		DonatedAt:      charlieDonationDate,
		VerifiedAt:     &charlieDonationDate,
		VerifiedBy:     &agency2.ID,
		NextEligibleAt: &charlieNextEligible,
		CreatedAt:      charlieDonationDate,
		UpdatedAt:      charlieDonationDate,
	}

	joanDonationDate1 := now.Add(-60 * 24 * time.Hour)
	joanNextEligible1 := joanDonationDate1.Add(56 * 24 * time.Hour)
	donationJoan1 := entities.Donation{
		ID:             primitive.NewObjectID(),
		DonorID:        donorJoan.ID,
		AgencyID:       agency1.ID,
		LocationID:     &loc1.ID,
		Pints:          2,
		Status:         entities.StatusVerified,
		DonatedAt:      joanDonationDate1,
		VerifiedAt:     &joanDonationDate1,
		VerifiedBy:     &agency1.ID,
		NextEligibleAt: &joanNextEligible1,
		CreatedAt:      joanDonationDate1,
		UpdatedAt:      joanDonationDate1,
	}

	joanDonationDate2 := now.Add(-120 * 24 * time.Hour)
	joanNextEligible2 := joanDonationDate2.Add(56 * 24 * time.Hour)
	donationJoan2 := entities.Donation{
		ID:             primitive.NewObjectID(),
		DonorID:        donorJoan.ID,
		AgencyID:       agency1.ID,
		LocationID:     &loc1.ID,
		Pints:          2,
		Status:         entities.StatusVerified,
		DonatedAt:      joanDonationDate2,
		VerifiedAt:     &joanDonationDate2,
		VerifiedBy:     &agency1.ID,
		NextEligibleAt: &joanNextEligible2,
		CreatedAt:      joanDonationDate2,
		UpdatedAt:      joanDonationDate2,
	}

	donationJoanPending := entities.Donation{
		ID:         primitive.NewObjectID(),
		DonorID:    donorJoan.ID,
		AgencyID:   agency1.ID,
		LocationID: &loc1.ID,
		Pints:      1,
		Status:     entities.StatusPending,
		DonatedAt:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	_, _ = mongoDB.Collection("donations").InsertMany(ctx, []interface{}{
		donationAlice1, donationAlice2, donationAlice3, donationAlice4, donationAlice5,
		donationBob1, donationBob2, donationBobPending, donationBobRejected,
		donationCharlie1, donationJoan1, donationJoan2, donationJoanPending,
	})

	statsAlice := entities.DonorStats{
		ID:             donorAlice.ID,
		TotalDonations: 5,
		TotalPints:     8,
		Points:         90,
		UpdatedAt:      time.Now().UTC(),
	}

	statsBob := entities.DonorStats{
		ID:             donorBob.ID,
		TotalDonations: 2,
		TotalPints:     4,
		Points:         40,
		UpdatedAt:      time.Now().UTC(),
	}

	statsCharlie := entities.DonorStats{
		ID:             donorCharlie.ID,
		TotalDonations: 1,
		TotalPints:     1,
		Points:         15,
		UpdatedAt:      time.Now().UTC(),
	}

	statsJoan := entities.DonorStats{
		ID:             donorJoan.ID,
		TotalDonations: 2,
		TotalPints:     4,
		Points:         40,
		UpdatedAt:      time.Now().UTC(),
	}

	_, _ = mongoDB.Collection("donor_stats").InsertMany(ctx, []interface{}{
		statsAlice, statsBob, statsCharlie, statsJoan,
	})

	fmt.Println("==================================================")
	fmt.Println("Seed script executed successfully!")
	fmt.Println("Created:")
	fmt.Printf("- Agency 1: %s (Password: password123)\n", agency1.Email)
	fmt.Printf("- Agency 2: %s (Password: password123)\n", agency2.Email)
	fmt.Printf("- Donor 1 (Alice): %s (Password: password123) - Leaderboard: 90 pts, Next Eligible: in %d days\n", donorAlice.Email, int(aliceNextEligible.Sub(time.Now()).Hours()/24))
	fmt.Printf("- Donor 2 (Bob): %s (Password: password123) - Leaderboard: 40 pts, 1 Pending, 1 Rejected\n", donorBob.Email)
	fmt.Printf("- Donor 3 (Charlie): %s (Password: password123) - Leaderboard: 15 pts, Eligible now\n", donorCharlie.Email)
	fmt.Printf("- Donor 4 (David): %s (Password: password123) - Never Donated\n", donorDavid.Email)
	fmt.Printf("- Custom Donor (Joan): %s (Password: assembly1234) - Leaderboard: 40 pts, 2 Verified, 1 Pending\n", donorJoan.Email)
	fmt.Println("==================================================")
}
