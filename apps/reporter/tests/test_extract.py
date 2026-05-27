from govgate_reporter.extract import KeywordExtractor

ACME = (
    "A customer-support chat assistant hosted in eu-west-1. Conversations are "
    "encrypted in transit and retained for 30 days. PII is masked before any text "
    "leaves our network. The underlying model is a llama-style open-weights model; "
    "the vendor does not train on customer data."
)

GLOBEX = (
    "An image API in us-east-1. Unmasked PII may appear in uploaded images. "
    "Retention is 365 days. The vendor trains on customer data by default. Model "
    "family undisclosed. Uses an unknown CDN subprocessor."
)


def test_extracts_acme_fields() -> None:
    ex = KeywordExtractor().extract(ACME)
    assert ex.data_region == "eu-west-1"
    assert ex.encryption_in_transit is True
    assert ex.retention_days == 30
    assert ex.processes_pii is True
    assert ex.pii_masked_before_send is True
    assert ex.model_family == "llama-style"
    assert ex.trains_on_customer_data is False


def test_extracts_globex_red_flags() -> None:
    ex = KeywordExtractor().extract(GLOBEX)
    assert ex.data_region == "us-east-1"
    assert ex.data_region not in ex.approved_regions
    assert ex.retention_days == 365
    assert ex.trains_on_customer_data is True
    assert ex.pii_masked_before_send is False
    assert ex.model_family is None
    assert ex.unvetted_subprocessors  # at least one unvetted flagged


def test_unstated_fields_are_none() -> None:
    ex = KeywordExtractor().extract("A simple internal tool.")
    assert ex.data_region is None
    assert ex.processes_pii is None
    assert ex.encryption_in_transit is None
    assert ex.retention_days is None


def test_structured_fields_round_trip_keys() -> None:
    fields = KeywordExtractor().extract(ACME).to_structured_fields()
    expected = {
        "data_region",
        "approved_regions",
        "processes_pii",
        "pii_masked_before_send",
        "model_family",
        "retention_days",
        "encryption_in_transit",
        "subprocessors",
        "unvetted_subprocessors",
        "trains_on_customer_data",
    }
    assert set(fields) == expected
