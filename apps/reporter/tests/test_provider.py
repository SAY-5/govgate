from govgate_reporter.provider import Extraction, FakeProvider


def test_fake_provider_exact_match_wins() -> None:
    p = FakeProvider()
    p.script("describes a chat tool", Extraction(model_family="gpt-style"))
    p.script("hello world", Extraction(model_family="llama-style"), exact=True)

    assert p.extract("hello world").model_family == "llama-style"


def test_fake_provider_substring_match() -> None:
    p = FakeProvider()
    p.script("eu-west", Extraction(data_region="eu-west-1"))
    assert p.extract("hosted in eu-west region").data_region == "eu-west-1"


def test_fake_provider_default_empty() -> None:
    p = FakeProvider()
    ex = p.extract("nothing scripted")
    assert ex == Extraction()


def test_to_structured_fields_shape() -> None:
    ex = Extraction(
        data_region="eu-west-1",
        processes_pii=True,
        pii_masked_before_send=True,
        retention_days=30,
    )
    fields = ex.to_structured_fields()
    assert fields["data_region"] == "eu-west-1"
    assert fields["processes_pii"] is True
    assert fields["retention_days"] == 30
    # unset optionals serialize as None, not missing
    assert fields["encryption_in_transit"] is None
    assert fields["model_family"] == ""
