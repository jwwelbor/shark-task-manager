"""Two independent observations of one defect, with candidate and report held fixed."""

def grouped_cases(identity, files, site):
    base = dict(file=site, line=1, severity='non-blocker', rule='ARCHITECTURE',
                summary='duplicate registration', diagnosis='policy rotation touches two registrations',
                impact='the maintainer must edit both registrations during rotation')
    packets = {label: dict(review_contract_version=2, candidate_identity=identity,
                           reviewed_files=files, findings=[]) for label in 'abcdef'}
    b = dict(base, evidence='rotation guide clause 3')
    d = dict(base, evidence='independent maintenance checklist')
    packets['b']['findings'] = [b]
    packets['d']['findings'] = [d]
    first = dict(b, origin='b', finding_id='F-1', disposition='triage', reason='two manual edits')
    second = dict(d, origin='d', finding_id='F-2', disposition='triage', reason='two manual edits')
    report = (f'Scope: inspected the frozen change.\n## B. Findings\n{site}:1 ARCHITECTURE: duplicate registration. '
              'B (rotation guide clause 3) and D (independent maintenance checklist) both observed two edits '
              'during rotation. Derive one registration from the other.\nVerdict: PASS-WITH-TRIAGE')
    for case, members, error in [
        ('both', [first, second], ''), ('reverse', [second, first], ''),
        ('missing-b', [second], 'angle b'), ('missing-d', [first], 'angle d'),
        ('duplicated-origin', [first, dict(second, origin='b')], 'angle'),
        ('swapped-support', [dict(first, evidence=d['evidence']), dict(second, evidence=b['evidence'])], 'angle b'),
        ('same-id', [first, dict(second, finding_id='F-1')], 'F-1'),
        ('unidentified', [dict(first, finding_id=''), second], 'angle b'),
        ('unattributed', [{k: v for k, v in first.items() if k != 'origin'}, second], 'angle b'),
        ('orphan', [first, second, dict(first, origin='c', finding_id='F-3')], 'angle c'),
        ('ambiguous-extra', [first, second, dict(base, evidence=b['evidence'], disposition='triage',
                                                  reason='duplicate', finding_id='F-3')], 'ambiguous'),
    ]:
        yield case, packets, report, dict(review_contract_version=2, candidate_identity=identity, findings=members), error
