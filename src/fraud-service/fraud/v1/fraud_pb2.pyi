import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PredictRequest(_message.Message):
    __slots__ = ("transaction_id", "tx_datetime", "customer_id", "terminal_id", "tx_time_seconds", "tx_time_days", "tx_amount", "tx_during_weekend", "tx_during_night", "customer_id_nb_tx_1day_window", "customer_id_avg_amount_1day_window", "customer_id_nb_tx_7day_window", "customer_id_avg_amount_7day_window", "customer_id_nb_tx_30day_window", "customer_id_avg_amount_30day_window", "terminal_id_nb_tx_1day_window", "terminal_id_risk_1day_window", "terminal_id_nb_tx_7day_window", "terminal_id_risk_7day_window", "terminal_id_nb_tx_30day_window", "terminal_id_risk_30day_window")
    TRANSACTION_ID_FIELD_NUMBER: _ClassVar[int]
    TX_DATETIME_FIELD_NUMBER: _ClassVar[int]
    CUSTOMER_ID_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_ID_FIELD_NUMBER: _ClassVar[int]
    TX_TIME_SECONDS_FIELD_NUMBER: _ClassVar[int]
    TX_TIME_DAYS_FIELD_NUMBER: _ClassVar[int]
    TX_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    TX_DURING_WEEKEND_FIELD_NUMBER: _ClassVar[int]
    TX_DURING_NIGHT_FIELD_NUMBER: _ClassVar[int]
    CUSTOMER_ID_NB_TX_1DAY_WINDOW_FIELD_NUMBER: _ClassVar[int]
    CUSTOMER_ID_AVG_AMOUNT_1DAY_WINDOW_FIELD_NUMBER: _ClassVar[int]
    CUSTOMER_ID_NB_TX_7DAY_WINDOW_FIELD_NUMBER: _ClassVar[int]
    CUSTOMER_ID_AVG_AMOUNT_7DAY_WINDOW_FIELD_NUMBER: _ClassVar[int]
    CUSTOMER_ID_NB_TX_30DAY_WINDOW_FIELD_NUMBER: _ClassVar[int]
    CUSTOMER_ID_AVG_AMOUNT_30DAY_WINDOW_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_ID_NB_TX_1DAY_WINDOW_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_ID_RISK_1DAY_WINDOW_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_ID_NB_TX_7DAY_WINDOW_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_ID_RISK_7DAY_WINDOW_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_ID_NB_TX_30DAY_WINDOW_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_ID_RISK_30DAY_WINDOW_FIELD_NUMBER: _ClassVar[int]
    transaction_id: int
    tx_datetime: _timestamp_pb2.Timestamp
    customer_id: int
    terminal_id: int
    tx_time_seconds: int
    tx_time_days: int
    tx_amount: float
    tx_during_weekend: int
    tx_during_night: int
    customer_id_nb_tx_1day_window: float
    customer_id_avg_amount_1day_window: float
    customer_id_nb_tx_7day_window: float
    customer_id_avg_amount_7day_window: float
    customer_id_nb_tx_30day_window: float
    customer_id_avg_amount_30day_window: float
    terminal_id_nb_tx_1day_window: float
    terminal_id_risk_1day_window: float
    terminal_id_nb_tx_7day_window: float
    terminal_id_risk_7day_window: float
    terminal_id_nb_tx_30day_window: float
    terminal_id_risk_30day_window: float
    def __init__(self, transaction_id: _Optional[int] = ..., tx_datetime: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., customer_id: _Optional[int] = ..., terminal_id: _Optional[int] = ..., tx_time_seconds: _Optional[int] = ..., tx_time_days: _Optional[int] = ..., tx_amount: _Optional[float] = ..., tx_during_weekend: _Optional[int] = ..., tx_during_night: _Optional[int] = ..., customer_id_nb_tx_1day_window: _Optional[float] = ..., customer_id_avg_amount_1day_window: _Optional[float] = ..., customer_id_nb_tx_7day_window: _Optional[float] = ..., customer_id_avg_amount_7day_window: _Optional[float] = ..., customer_id_nb_tx_30day_window: _Optional[float] = ..., customer_id_avg_amount_30day_window: _Optional[float] = ..., terminal_id_nb_tx_1day_window: _Optional[float] = ..., terminal_id_risk_1day_window: _Optional[float] = ..., terminal_id_nb_tx_7day_window: _Optional[float] = ..., terminal_id_risk_7day_window: _Optional[float] = ..., terminal_id_nb_tx_30day_window: _Optional[float] = ..., terminal_id_risk_30day_window: _Optional[float] = ...) -> None: ...

class PredictResponse(_message.Message):
    __slots__ = ("is_fraud", "fraud_probability")
    IS_FRAUD_FIELD_NUMBER: _ClassVar[int]
    FRAUD_PROBABILITY_FIELD_NUMBER: _ClassVar[int]
    is_fraud: bool
    fraud_probability: float
    def __init__(self, is_fraud: _Optional[bool] = ..., fraud_probability: _Optional[float] = ...) -> None: ...
