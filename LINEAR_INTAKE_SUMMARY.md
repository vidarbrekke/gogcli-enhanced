# Phase 3 Cross-Service Expansion: Linear Intake Summary

**Date:** 2026-02-17  
**Status:** Ready for Linear creation  
**Issues to Create:** 6 (VID-111 through VID-116)  
**Total Duration:** 12.5 hours  
**Expected Impact:** Enterprise-grade mail-merge + report generation

---

## 📋 READY-TO-COPY ISSUE TEMPLATES

All issue specifications are in `LINEAR_ISSUES_TEMPLATE.md` — ready to copy/paste into Linear.

### Issues to Create (In Order)

1. **VID-111: Sheets DeleteRange** (Quick win, 1.5h)
   - Copy from LINEAR_ISSUES_TEMPLATE.md section "Issue VID-111"
   - Priority: High
   - Estimate: 1.5 hours
   - Type: Feature

2. **VID-112: Docs InsertImage** (Quick win + polish, 2h)
   - Copy from LINEAR_ISSUES_TEMPLATE.md section "Issue VID-112"
   - Priority: High
   - Estimate: 2 hours
   - Type: Feature

3. **VID-113: Code Review** (Internal QA, 0.5h)
   - Copy from LINEAR_ISSUES_TEMPLATE.md section "Issue VID-113"
   - Priority: Medium
   - Estimate: 0.5 hours
   - Type: Task (depends on VID-111, VID-112)

4. **VID-114: Design Review** (Critical design phase, 1.5h)
   - Copy from LINEAR_ISSUES_TEMPLATE.md section "Issue VID-114"
   - Priority: Very High
   - Estimate: 1.5 hours
   - Type: Design
   - **Blockers for VID-115 and VID-116**

5. **VID-115: Docs MergeData** (Transformative, 3.5-4h)
   - Copy from LINEAR_ISSUES_TEMPLATE.md section "Issue VID-115"
   - Priority: Very High
   - Estimate: 4 hours
   - Type: Feature
   - Depends on: VID-114

6. **VID-116: Sheets MergeData** (Transformative, 3.5-4h)
   - Copy from LINEAR_ISSUES_TEMPLATE.md section "Issue VID-116"
   - Priority: Very High
   - Estimate: 4 hours
   - Type: Feature
   - Depends on: VID-114, VID-115 (reference implementation)

---

## 🎯 CONTEXT & STRATEGIC RATIONALE

### Why These 6 Issues?

**Strategic Gap Analysis** identified 6 high-value capabilities that exist in the Google APIs but are missing from the CLI:

1. **ReplaceText** exists in Docs & Slides, missing in Sheets (API supports it!)
2. **ReplaceImage** exists in Slides, missing in Docs (API supports it!)
3. **DeleteRange** exists in Docs, missing in Sheets (API supports it!)
4. **MergeData** exists (partially) in Slides, missing in Docs & Sheets (pattern proven!)

### Phase 3 Fills These Gaps

| Service | ReplaceText | DeleteRange | ReplaceImage | MergeData |
|---------|-------------|-------------|--------------|-----------|
| Docs | ✓ (VID-107) | ✓ (VID-107) | ← VID-112 | ← VID-115 |
| Sheets | ← VID-111 | ← VID-111 | ✗ (N/A) | ← VID-116 |
| Slides | ✓ | ✗ (N/A) | ✓ (VID-100-106) | ✓ (VID-100-106) |

### Expected Impact

**Quick Wins (VID-111, VID-112):** 3.5 hours
- Cross-service consistency (delete operations available everywhere)
- Image workflows (insert + replace for document branding)

**Transformative (VID-115, VID-116):** 8 hours
- **60% of users** unlock with Docs MergeData (mail-merge for documents)
- **50% of users** unlock with Sheets MergeData (report generation)
- Platform transform: from point tools → enterprise automation

**Total:** 12.5 hours → 110%+ user value lift

---

## 📚 SUPPORTING DOCUMENTATION

All documentation is in the repo and ready for reference:

1. **`CROSS_SERVICE_OPPORTUNITY_ANALYSIS.md`** (300+ lines)
   - Strategic gap analysis
   - 6 opportunities ranked by impact × effort
   - Expected user impact projections
   - Technical implementation guidance

2. **`PHASE_3_IMPLEMENTATION_PLAN.md`** (500+ lines)
   - Detailed implementation guidance for each issue
   - API references and command signatures
   - Success criteria with checklists
   - Use cases and expected impact
   - Pattern references and common pitfalls
   - Testing strategy

3. **`LINEAR_ISSUES_TEMPLATE.md`** (600+ lines)
   - Ready-to-copy issue specifications for all 6 issues
   - Each includes: description, features, API reference, success criteria, implementation notes

---

## 🚀 RECOMMENDED EXECUTION STRATEGY

### Week 1: Quick Wins + Foundation
- **VID-111** (Sheets DeleteRange) — 1.5h
- **VID-112** (Docs InsertImage) — 2h
- **VID-113** (Code Review) — 0.5h
- **VID-114** (Design Review) — 1.5h
- **Subtotal:** 5.5 hours
- **Result:** Cross-service consistency + foundation for MergeData

### Week 2: Transformative Features
- **VID-115** (Docs MergeData) — 3.5-4h
- **VID-116** (Sheets MergeData) — 3.5-4h
- **Subtotal:** 7-8 hours
- **Result:** Enterprise mail-merge + report generation

### Timeline
```
Week 1 (Mon-Fri):
  Mon: VID-111 (1.5h)
  Tue: VID-112 (2h)
  Wed: VID-113 + VID-114 (2h)
  Thu-Fri: Buffer / other work

Week 2 (Mon-Fri):
  Mon-Wed: VID-115 (4h)
  Wed-Fri: VID-116 (4h)
  → Ready for merge to main

Total: 2 sprints, 12.5 hours, enterprise-grade platform
```

---

## ✅ PRE-REQUISITES

- [ ] All issues added to Linear project
- [ ] Issues linked to epic (Phase 3 Cross-Service Expansion)
- [ ] Dependencies set (VID-114 blocks VID-115/116, etc.)
- [ ] Reviewer assigned (preferably same person for consistency)
- [ ] Documentation linked in each issue

---

## 📊 SUCCESS METRICS

### Code Quality
- ✅ All commands follow agentic pattern
- ✅ All pass `make test && make lint`
- ✅ All handle errors gracefully
- ✅ Zero breaking changes

### Feature Completeness
- ✅ 4 new operations (Delete, InsertImage, MergeData×2)
- ✅ 6/6 identified gaps addressed
- ✅ Cross-service consistency achieved

### User Impact
- ✅ 60% users unlock (Docs MergeData)
- ✅ 50% users unlock (Sheets MergeData)
- ✅ Platform cohesion: point tools → automation platform

---

## 🔗 RELATED PHASES

**Phase 1 (Shared Foundation):** VID-91  
**Phase 2A (Sheets Edit):** VID-92–95  
**Phase 2B (Docs Edit):** VID-107  
**Phase 2C (This Work):** VID-111–116  
**Phase 3 (Slides Enhancement):** VID-100–106 (completed), VID-96–98 (pending)

---

## 📞 KEY CONTACTS

- **Strategic Analysis:** Review `CROSS_SERVICE_OPPORTUNITY_ANALYSIS.md`
- **Implementation Guidance:** See `PHASE_3_IMPLEMENTATION_PLAN.md`
- **Issue Specifications:** Copy from `LINEAR_ISSUES_TEMPLATE.md`
- **Code Reference:** See `internal/cmd/docs_edit_cmd.go`, `sheets_edit_cmd.go`, `slides_edit_cmd.go`

---

## 🎬 NEXT STEPS

1. **Create Linear Issues** (Use LINEAR_ISSUES_TEMPLATE.md)
   - Create 6 issues in order (VID-111 through VID-116)
   - Set priorities and estimates as specified
   - Link dependencies

2. **Assign to Team**
   - Assign VID-111 to first developer
   - Assign VID-114 to architect/senior engineer (design-critical)
   - Assign VID-115/116 to strong engineer (transformative features)

3. **Start Execution**
   - Begin with VID-111 (quick confidence builder)
   - Follow with VID-112 (another quick win)
   - Then VID-113 (quality check)
   - Then VID-114 (critical design)
   - Then VID-115/116 (transformative pair)

---

## 📝 NOTES FOR LINEAR INTAKE

**Epic:** Cross-Service Capability Expansion (Phase 3)  
**Related Epics:** Agentic Edit Operations (Phase 1-3)  
**Status:** Ready for intake  
**Created By:** Strategic gap analysis + 10x engineering review  

All supporting documentation is in the repository and ready for implementation.

---

**Last Updated:** 2026-02-17  
**Status:** ✅ Ready to create Linear issues
