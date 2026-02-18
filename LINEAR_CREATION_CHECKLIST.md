# Linear Issues Creation Checklist — Phase 3 (VID-111 through VID-116)

**Instructions:** Create these 6 issues in Linear in the order listed below. Copy the specifications from `LINEAR_ISSUES_TEMPLATE.md`.

---

## 📋 Issue Creation Workflow

### Pre-Requisites
- [ ] Linear project open
- [ ] Team/Project selected
- [ ] Have `LINEAR_ISSUES_TEMPLATE.md` open for reference

---

## 🟢 ISSUE 1: VID-111 - Sheets DeleteRange

**Status:** Create this first (no dependencies)

**In Linear, create:**
- [ ] **Title:** Sheets Edit: DeleteRange - Delete cell range with shift options
- [ ] **Priority:** High
- [ ] **Team:** Engineering
- [ ] **Estimate:** 1.5 hours (or 1 point if using points)
- [ ] **Type/Status:** Todo (Feature)
- [ ] **Description:** Copy from `LINEAR_ISSUES_TEMPLATE.md` section "Issue VID-111"
- [ ] **Labels:** Add label for "quick-win" or "Phase 3"
- [ ] **Related Issues:** Link to VID-112, VID-112 (other quick wins in phase)

**Key Details to Include:**
```
Command: gog sheets edit delete-range <spreadsheetId> \
  --range "A1:C10" \
  --shift-dimension ROWS|COLUMNS
```

---

## 🟢 ISSUE 2: VID-112 - Docs InsertImage

**Status:** Parallel with VID-111 (no dependencies)

**In Linear, create:**
- [ ] **Title:** Docs Edit: InsertImage - Insert image at specific position
- [ ] **Priority:** High
- [ ] **Team:** Engineering
- [ ] **Estimate:** 2 hours
- [ ] **Type/Status:** Todo (Feature)
- [ ] **Description:** Copy from `LINEAR_ISSUES_TEMPLATE.md` section "Issue VID-112"
- [ ] **Labels:** "quick-win", "Phase 3"
- [ ] **Related Issues:** Link to VID-111, VID-4893e06 (ReplaceImage)

**Key Details to Include:**
```
Command: gog docs edit insert-image <docId> \
  --uri "https://example.com/logo.png" \
  --index 1
```

---

## 🔵 ISSUE 3: VID-113 - Code Review

**Status:** Depends on VID-111 & VID-112

**In Linear, create:**
- [ ] **Title:** Code Review & Testing - Phase 3 Quick Wins (VID-111, VID-112)
- [ ] **Priority:** Medium
- [ ] **Team:** Engineering
- [ ] **Estimate:** 0.5 hours
- [ ] **Type/Status:** Todo (Task)
- [ ] **Description:** Copy from `LINEAR_ISSUES_TEMPLATE.md` section "Issue VID-113"
- [ ] **Labels:** "code-review", "Phase 3"
- [ ] **Depends On:** VID-111, VID-112

**Mark as blocking:** Nothing (unblocks nothing by itself)

---

## 🟠 ISSUE 4: VID-114 - Design Review (CRITICAL)

**Status:** Must complete before VID-115 & VID-116

**In Linear, create:**
- [ ] **Title:** Design Review - Mail-Merge Pattern for Docs & Sheets
- [ ] **Priority:** Very High
- [ ] **Team:** Engineering
- [ ] **Estimate:** 1.5 hours
- [ ] **Type/Status:** Todo (Design)
- [ ] **Description:** Copy from `LINEAR_ISSUES_TEMPLATE.md` section "Issue VID-114"
- [ ] **Labels:** "design", "critical", "Phase 3"
- [ ] **Depends On:** None (but is prerequisite for VID-115, VID-116)

**Mark as blocked by:** Nothing  
**Mark as blocking:** VID-115, VID-116

---

## 🔴 ISSUE 5: VID-115 - Docs MergeData (TRANSFORMATIVE)

**Status:** Depends on VID-114, can reference Slides implementation

**In Linear, create:**
- [ ] **Title:** Docs Edit: MergeData - Mail-merge documents from template + data
- [ ] **Priority:** Very High
- [ ] **Team:** Engineering
- [ ] **Estimate:** 4 hours
- [ ] **Type/Status:** Todo (Feature)
- [ ] **Description:** Copy from `LINEAR_ISSUES_TEMPLATE.md` section "Issue VID-115"
- [ ] **Labels:** "transformative", "mail-merge", "Phase 3"
- [ ] **Depends On:** VID-114 (design complete)
- [ ] **Related Issues:** VID-100-106 (Slides MergeData reference)

**Command Preview:**
```
gog docs edit merge-data <templateId> \
  --data-file employees.json \
  --filename-format "Offer Letter - {{firstName}}"
```

---

## 🔴 ISSUE 6: VID-116 - Sheets MergeData (TRANSFORMATIVE)

**Status:** Depends on VID-114, can reference VID-115

**In Linear, create:**
- [ ] **Title:** Sheets Edit: MergeData - Mail-merge spreadsheets from template + data
- [ ] **Priority:** Very High
- [ ] **Team:** Engineering
- [ ] **Estimate:** 4 hours
- [ ] **Type/Status:** Todo (Feature)
- [ ] **Description:** Copy from `LINEAR_ISSUES_TEMPLATE.md` section "Issue VID-116"
- [ ] **Labels:** "transformative", "mail-merge", "Phase 3"
- [ ] **Depends On:** VID-114 (design complete), VID-115 (reference implementation)
- [ ] **Related Issues:** VID-100-106 (Slides MergeData reference)

**Command Preview:**
```
gog sheets edit merge-data <templateId> \
  --data-file reports.json \
  --filename-format "Report - {{quarter}}"
```

---

## 📊 Dependency Graph

```
VID-111 (Sheets DeleteRange) ──→ VID-113 (Code Review)
                                     ↑
VID-112 (Docs InsertImage)  ────────↑


VID-114 (Design Review) ──→ VID-115 (Docs MergeData) ──→ VID-116 (Sheets MergeData)
                             (reference)
```

---

## ✅ Verification Checklist

After creating all 6 issues:

- [ ] VID-111: Created, High priority, 1.5h estimate
- [ ] VID-112: Created, High priority, 2h estimate
- [ ] VID-113: Created, Medium priority, depends on VID-111 & VID-112
- [ ] VID-114: Created, Very High priority, BLOCKING VID-115 & VID-116
- [ ] VID-115: Created, Very High priority, depends on VID-114
- [ ] VID-116: Created, Very High priority, depends on VID-114
- [ ] All issues have labels: "Phase 3"
- [ ] All issues linked to `LINEAR_ISSUES_TEMPLATE.md` in description (reference for details)
- [ ] Dependencies configured correctly
- [ ] Total estimate: ~12.5 hours

---

## 📐 Quick Reference: Issue Properties Summary

| VID | Title | Hours | Priority | Type | Depends On | Blocks |
|-----|-------|-------|----------|------|-----------|--------|
| 111 | Sheets DeleteRange | 1.5 | High | Feature | None | VID-113 |
| 112 | Docs InsertImage | 2 | High | Feature | None | VID-113 |
| 113 | Code Review | 0.5 | Medium | Task | 111, 112 | None |
| 114 | Design Review | 1.5 | Very High | Design | None | 115, 116 |
| 115 | Docs MergeData | 4 | Very High | Feature | VID-114 | None |
| 116 | Sheets MergeData | 4 | Very High | Feature | VID-114 | None |

---

## 🔗 Reference Documents

When creating issues, link to these supporting docs:
- `CROSS_SERVICE_OPPORTUNITY_ANALYSIS.md` — Strategic context
- `PHASE_3_IMPLEMENTATION_PLAN.md` — Implementation details
- `LINEAR_ISSUES_TEMPLATE.md` — Full specifications

---

## 📝 Pro Tips for Linear

1. **Copy-paste the Description:** The descriptions in `LINEAR_ISSUES_TEMPLATE.md` are formatted for Linear Markdown. Just copy the entire description section for each issue.

2. **Set Estimates Correctly:**
   - VID-111: 1.5h
   - VID-112: 2h
   - VID-113: 0.5h
   - VID-114: 1.5h (design, not coding)
   - VID-115: 4h
   - VID-116: 4h
   - **Total: 12.5h**

3. **Link Dependencies:**
   - VID-113 should "depend on" VID-111 and VID-112
   - VID-115 should "depend on" VID-114
   - VID-116 should "depend on" VID-114

4. **Add Labels:** Suggest labels:
   - "Phase-3" (all issues)
   - "quick-win" (VID-111, 112)
   - "transformative" (VID-115, 116)
   - "critical-design" (VID-114)

---

## ⏭️ After Creating Issues

Once all 6 issues are created:

1. [ ] Share the Linear epic/cycle with the team
2. [ ] Assign team members (see below)
3. [ ] Link to supporting documentation in project description
4. [ ] Post in team chat: "Phase 3 Planning Complete: 6 issues ready for intake"

### Recommended Assignments

- **VID-111** → Senior backend engineer (quick win to build momentum)
- **VID-112** → Mid-level engineer (straightforward feature)
- **VID-113** → Code reviewer (can be anyone, 0.5h internal)
- **VID-114** → Architect/senior engineer (design-critical, must be thoughtful)
- **VID-115** → Strong engineer (transformative, complex)
- **VID-116** → Strong engineer (transformative, complex, pairs well with VID-115)

---

## 📞 Questions During Creation?

Refer to:
- **"What should this command look like?"** → See `LINEAR_ISSUES_TEMPLATE.md`
- **"What's the strategic reason?"** → See `CROSS_SERVICE_OPPORTUNITY_ANALYSIS.md`
- **"How do I implement this?"** → See `PHASE_3_IMPLEMENTATION_PLAN.md`
- **"What are success criteria?"** → Each issue in Linear_ISSUES_TEMPLATE included checklist

---

**Status: Ready to create all 6 issues**  
**Estimated Creation Time: 15-20 minutes**  
**Ready to Execute: Once all issues created and assigned**
