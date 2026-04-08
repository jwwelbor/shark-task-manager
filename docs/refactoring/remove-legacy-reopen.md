The blocker is that cascadeParentReopens always looks up a feature to get feature.EpicID, but when
  called from CreateFeature we already have the epic. The cascade just needs a way to start from an epic directly.

  The problem in one line:
  // cascadeTrigger only has featureID — no epicID
  epic, err = deps.epicRepo.GetByID(ctx, feature.EpicID) // always goes through feature

  The fix: add epicID to cascadeTrigger so the epic-only leg can skip the feature lookup:

  type cascadeTrigger struct {
      triggerKey  string
      triggerKind string
      triggerType models.EntityType
      startLeg    cascadeLeg
      featureID   int64  // used when startLeg == cascadeLegFeature
      epicID      int64  // NEW: used when startLeg == cascadeLegEpic, bypasses feature lookup
  }

  Then in cascadeParentReopens, Phase 1:
  if trigger.startLeg == cascadeLegEpic && trigger.epicID != 0 {
      // We already know the epic — skip the feature lookup entirely
      epic, err = deps.epicRepo.GetByID(ctx, trigger.epicID)
  } else {
      feature, err = deps.featureRepo.GetByID(ctx, trigger.featureID)
      // ... existing logic ...
      epic, err = deps.epicRepo.GetByID(ctx, feature.EpicID)
  }

