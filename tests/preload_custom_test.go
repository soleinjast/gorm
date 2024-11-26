package tests_test

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Define models
type Item struct {
	ID        uint
	Name      string
	Tags      []Tag `gorm:"many2many:item_tags"`
	CreatedAt time.Time
}

type Tag struct {
	ID        uint
	Name      string
	Status    string
	SubTags   []SubTag `gorm:"many2many:tag_sub_tags"`
}

type SubTag struct {
	ID     uint
	Name   string
	Status string
}

// Setup the in-memory SQLite database
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	err = db.AutoMigrate(&Item{}, &Tag{}, &SubTag{})
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}
	return db
}

// Test default preloading functionality
func TestDefaultPreload(t *testing.T) {
	db := setupTestDB(t)

	tag1 := Tag{Name: "Tag1", Status: "active"}
	item := Item{Name: "Item1", Tags: []Tag{tag1}}
	db.Create(&item)

	var items []Item
	err := db.Preload("Tags").Find(&items).Error
	if err != nil {
		t.Fatalf("default preload failed: %v", err)
	}

	if len(items) != 1 || len(items[0].Tags) != 1 || items[0].Tags[0].Name != "Tag1" {
		t.Errorf("unexpected results in TestDefaultPreload: %v", items)
	}
}

// Test preloading with custom SQL joins and conditions
func TestCustomJoinsWithConditions(t *testing.T) {
	db := setupTestDB(t)

	tag1 := Tag{Name: "Tag1", Status: "active"}
	tag2 := Tag{Name: "Tag2", Status: "inactive"}
	item := Item{Name: "Item1", Tags: []Tag{tag1, tag2}}
	db.Create(&item)

	var items []Item
	err := db.Preload("Tags", func(tx *gorm.DB) *gorm.DB {
		return tx.Joins("JOIN item_tags ON item_tags.tag_id = tags.id").
			Where("tags.status = ?", "active")
	}).Find(&items).Error
	if err != nil {
		t.Fatalf("custom join with conditions failed: %v", err)
	}

	if len(items) != 1 || len(items[0].Tags) != 1 || items[0].Tags[0].Status != "active" {
		t.Errorf("unexpected results in TestCustomJoinsWithConditions: %v", items)
	}
}

// Test nested preloading with custom joins
func TestNestedPreloadWithCustomJoins(t *testing.T) {
	db := setupTestDB(t)

	subTag := SubTag{Name: "SubTag1", Status: "active"}
	tag := Tag{Name: "Tag1", Status: "active", SubTags: []SubTag{subTag}}
	item := Item{Name: "Item1", Tags: []Tag{tag}}
	db.Create(&item)

	var items []Item
	err := db.Preload("Tags.SubTags", func(tx *gorm.DB) *gorm.DB {
		return tx.Joins("JOIN tag_sub_tags ON tag_sub_tags.sub_tag_id = sub_tags.id").
			Where("sub_tags.status = ?", "active")
	}).Find(&items).Error
	if err != nil {
		t.Fatalf("nested preload with custom joins failed: %v", err)
	}

	if len(items) != 1 || len(items[0].Tags) != 1 || len(items[0].Tags[0].SubTags) != 1 || items[0].Tags[0].SubTags[0].Name != "SubTag1" {
		t.Errorf("unexpected results in TestNestedPreloadWithCustomJoins: %v", items)
	}
}

// Test behavior when no matching records exist
func TestNoMatchingRecords(t *testing.T) {
	db := setupTestDB(t)

	tag := Tag{Name: "Tag1", Status: "inactive"}
	item := Item{Name: "Item1", Tags: []Tag{tag}}
	db.Create(&item)

	var items []Item
	err := db.Preload("Tags", func(tx *gorm.DB) *gorm.DB {
		return tx.Joins("JOIN item_tags ON item_tags.tag_id = tags.id").
			Where("tags.status = ?", "active")
	}).Find(&items).Error
	if err != nil {
		t.Fatalf("preload with no matching records failed: %v", err)
	}

	if len(items) != 1 || len(items[0].Tags) != 0 {
		t.Errorf("unexpected results in TestNoMatchingRecords: %v", items)
	}
}

// Test behavior with an empty database
func TestEmptyDatabase(t *testing.T) {
	db := setupTestDB(t)

	var items []Item
	err := db.Preload("Tags").Find(&items).Error
	if err != nil {
		t.Fatalf("preload with empty database failed: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("unexpected results in TestEmptyDatabase: %v", items)
	}
}

// Test multiple items with different tag statuses
func TestMultipleItemsWithDifferentTagStatuses(t *testing.T) {
	db := setupTestDB(t)

	tag1 := Tag{Name: "Tag1", Status: "active"}
	tag2 := Tag{Name: "Tag2", Status: "inactive"}
	item1 := Item{Name: "Item1", Tags: []Tag{tag1}}
	item2 := Item{Name: "Item2", Tags: []Tag{tag2}}
	db.Create(&item1)
	db.Create(&item2)

	var items []Item
	err := db.Preload("Tags", func(tx *gorm.DB) *gorm.DB {
		return tx.Joins("JOIN item_tags ON item_tags.tag_id = tags.id").
			Where("tags.status = ?", "active")
	}).Find(&items).Error
	if err != nil {
		t.Fatalf("preload with multiple items failed: %v", err)
	}

	if len(items) != 2 || len(items[0].Tags) != 1 || len(items[1].Tags) != 0 {
		t.Errorf("unexpected results in TestMultipleItemsWithDifferentTagStatuses: %v", items)
	}
}

// Test duplicate preload conditions
func TestDuplicatePreloadConditions(t *testing.T) {
	db := setupTestDB(t)

	tag1 := Tag{Name: "Tag1", Status: "active"}
	tag2 := Tag{Name: "Tag2", Status: "inactive"}
	item := Item{Name: "Item1", Tags: []Tag{tag1, tag2}}
	db.Create(&item)

	var activeTagsItems []Item
	var inactiveTagsItems []Item

	err := db.Preload("Tags", func(tx *gorm.DB) *gorm.DB {
		return tx.Where("status = ?", "active")
	}).Find(&activeTagsItems).Error
	if err != nil {
		t.Fatalf("preload for active tags failed: %v", err)
	}

	err = db.Preload("Tags", func(tx *gorm.DB) *gorm.DB {
		return tx.Where("status = ?", "inactive")
	}).Find(&inactiveTagsItems).Error
	if err != nil {
		t.Fatalf("preload for inactive tags failed: %v", err)
	}

	if len(activeTagsItems) != 1 || len(activeTagsItems[0].Tags) != 1 || activeTagsItems[0].Tags[0].Status != "active" {
		t.Errorf("unexpected active tag results in TestDuplicatePreloadConditions: %v", activeTagsItems)
	}

	if len(inactiveTagsItems) != 1 || len(inactiveTagsItems[0].Tags) != 1 || inactiveTagsItems[0].Tags[0].Status != "inactive" {
		t.Errorf("unexpected inactive tag results in TestDuplicatePreloadConditions: %v", inactiveTagsItems)
	}
}