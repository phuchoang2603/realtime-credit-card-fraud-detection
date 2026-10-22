from datetime import date

from simulator.generation import build_history
from simulator.profiles import (
    CARD_TESTING,
    HISTORY_STAGE,
    STOLEN_CARD_NEW_DEVICE,
    build_profiles,
    compromised_on,
    schedule_for_day,
    stage_rng,
)
from tests.conftest import small_config


def _histories(config, day):
    profiles = build_profiles(config)
    schedule = schedule_for_day(config, day, profiles)
    random = stage_rng(config.seed, HISTORY_STAGE, 0, 0)
    return schedule, [
        build_history(config, profiles, schedule, index, day, index, random) for index in range(len(schedule))
    ]


def test_profiles_are_reproducible_from_the_seed():
    assert build_profiles(small_config()).digest() == build_profiles(small_config()).digest()
    assert build_profiles(small_config()).digest() != build_profiles(small_config(seed=8)).digest()


def test_compromise_windows_started_earlier_continue_on_a_rebuilt_day():
    config = small_config(**{"scenarios.stolen_card_new_device.rate": 0.2})
    profiles = build_profiles(config)
    started = set(compromised_on(config, STOLEN_CARD_NEW_DEVICE, 0).tolist())
    schedule = schedule_for_day(config, date(2026, 7, 3), profiles)
    fraud = schedule.scenarios == STOLEN_CARD_NEW_DEVICE
    from_day_zero = set(schedule.customers[fraud & (schedule.groups == 0)].tolist())
    assert from_day_zero and from_day_zero <= started
    assert not (schedule.groups[fraud] < 0).any()


def test_card_testing_bursts_share_a_device_across_many_low_amount_methods():
    config = small_config(**{"scenarios.card_testing.rate": 3})
    schedule, histories = _histories(config, date(2026, 7, 2))
    bursts = {}
    for index, history in enumerate(histories):
        if schedule.scenarios[index] != CARD_TESTING:
            continue
        created = history.events[0]["payment_created"]["snapshot"]
        signals = history.events[1]["attempt_started"]["signals"]
        assert created["total"]["minor_units"] <= 500
        bursts.setdefault(int(schedule.groups[index]), []).append(signals)
    assert bursts
    for signals in bursts.values():
        assert len({signal["device_fingerprint"] for signal in signals}) == 1
        assert len({signal["payment_method"]["fingerprint"] for signal in signals}) == len(signals)
        span = max(signal["attempted_at"] for signal in signals) - min(signal["attempted_at"] for signal in signals)
        assert span.total_seconds() <= config.scenarios.card_testing.burst_minutes * 60 + 5


def test_disabled_scenarios_never_appear_in_labels():
    config = small_config(
        **{
            "scenarios.card_testing.rate": 0,
            "scenarios.account_takeover.rate": 0,
            "scenarios.friendly_fraud.rate": 0,
        }
    )
    _, histories = _histories(config, date(2026, 7, 3))
    scenarios = {label["scenario"] for history in histories for label in history.labels}
    assert not scenarios & {"card_testing", "account_takeover", "friendly_fraud"}
